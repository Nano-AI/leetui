// Local test driver for LeetCode JavaScript and TypeScript solutions.
//
// Adapted in spirit from leetgo (https://github.com/j178/leetgo), MIT licensed,
// Copyright (c) 2022 Jo. leetgo has no JS driver; this follows the same contract
// its Python one does so the two behave identically from Go's side.
//
// HOW THIS IS RUN
// ---------------
// leetui concatenates this file, the user's solution, and a small invocation
// into one file and hands it to `node`. Concatenation rather than import
// because LeetCode's starter code declares a bare `var twoSum = function(...)`
// with no exports, and the judge supplies ListNode/TreeNode as globals — which
// is exactly what this file is, one scope up.
//
// It also means TypeScript costs nothing: Node strips the types across the
// whole file, so solution.ts runs the same way solution.js does.
//
// Every name here is prefixed so it cannot collide with a solution's own
// variables. `ListNode` and `TreeNode` are the two deliberate exceptions —
// LeetCode's snippets reference them by those names and will not run without.

/** LeetCode's singly-linked list node. */
function ListNode(val, next) {
  this.val = val === undefined ? 0 : val;
  this.next = next === undefined ? null : next;
}

/** LeetCode's binary tree node. */
function TreeNode(val, left, right) {
  this.val = val === undefined ? 0 : val;
  this.left = left === undefined ? null : left;
  this.right = right === undefined ? null : right;
}

const __leetui = {
  buildList(values) {
    let head = null;
    for (let i = values.length - 1; i >= 0; i--) head = new ListNode(values[i], head);
    return head;
  },

  dumpList(node) {
    const out = [];
    const seen = new Set();
    while (node !== null && node !== undefined) {
      // A deliberate cycle (linked list cycle II) would otherwise hang the
      // runner rather than fail it.
      if (seen.has(node)) { out.push('"...cycle"'); break; }
      seen.add(node);
      out.push(JSON.stringify(node.val));
      node = node.next;
    }
    return "[" + out.join(",") + "]";
  },

  /** Build from LeetCode's level-order form, where null marks a gap. */
  buildTree(values) {
    if (!values.length || values[0] === null) return null;
    const root = new TreeNode(values[0]);
    const queue = [root];
    let i = 1;
    while (queue.length && i < values.length) {
      const node = queue.shift();
      if (i < values.length) {
        if (values[i] !== null) { node.left = new TreeNode(values[i]); queue.push(node.left); }
        i++;
      }
      if (i < values.length) {
        if (values[i] !== null) { node.right = new TreeNode(values[i]); queue.push(node.right); }
        i++;
      }
    }
    return root;
  },

  /** Level-order with trailing nulls trimmed, matching what the judge prints. */
  dumpTree(root) {
    const out = [];
    const queue = [root];
    while (queue.length) {
      const node = queue.shift();
      if (node === null || node === undefined) { out.push("null"); continue; }
      out.push(JSON.stringify(node.val));
      queue.push(node.left);
      queue.push(node.right);
    }
    while (out.length && out[out.length - 1] === "null") out.pop();
    return "[" + out.join(",") + "]";
  },

  /** Turn one line of test input into a value, using metaData's own spelling. */
  parse(type, raw) {
    const text = raw.trim();
    if (type === "ListNode") return __leetui.buildList(JSON.parse(text));
    if (type === "TreeNode") return __leetui.buildTree(JSON.parse(text));
    if (type === "ListNode[]") return JSON.parse(text).map(__leetui.buildList);
    if (type === "TreeNode[]") return JSON.parse(text).map(__leetui.buildTree);
    // Everything else is plain JSON. "character" arrives as a 1-char string.
    return JSON.parse(text);
  },

  /**
   * Render a return value the way the judge prints it.
   *
   * `type` is the declared return type, and it matters for exactly one case: an EMPTY
   * list or tree. The value is null either way, but the judge prints `[]` for a
   * ListNode and `null` for an integer, and nothing about the value itself can tell
   * those apart. Reversing an empty list is the case that finds this.
   */
  dump(value, type) {
    if (value instanceof ListNode) return __leetui.dumpList(value);
    if (value instanceof TreeNode) return __leetui.dumpTree(value);
    if (Array.isArray(value)) return "[" + value.map((v) => __leetui.dump(v)).join(",") + "]";
    if (value === null || value === undefined) {
      return (type === "ListNode" || type === "TreeNode") ? "[]" : "null";
    }
    return JSON.stringify(value);
  },

  stdin() {
    const fs = require("fs");
    return fs.readFileSync(0, "utf8").split("\n");
  },

  /**
   * Read one test case, run it, print the result.
   *
   * `mutates` names the parameter to print instead of the return value, for
   * problems whose answer is the modified input. leetui passes it from its
   * override table; metaData cannot express it.
   */
  run(fn, types, mutates, returnType) {
    const lines = __leetui.stdin();
    if (lines.length < types.length) {
      console.error(`expected ${types.length} input lines, got ${lines.length}`);
      process.exit(1);
    }
    const args = types.map((t, i) => __leetui.parse(t, lines[i]));
    const result = fn(...args);

    if (mutates !== null && mutates !== undefined) {
      const arg = args[mutates];
      // In-place problems return the new length and expect the prefix.
      if (typeof result === "number" && Array.isArray(arg)) {
        console.log(__leetui.dump(arg.slice(0, result)));
      } else {
        console.log(__leetui.dump(arg));
      }
      return;
    }
    console.log(__leetui.dump(result, returnType));
  },

  /**
   * Run a design problem: a constructor, then a sequence of operations.
   *
   *     ["LRUCache","put","get"]
   *     [[2],[1,1],[1]]
   *
   * Output is one list of each call's return value, with null for the
   * constructor and for methods that return nothing.
   */
  runDesign(ctor, typesByMethod) {
    const lines = __leetui.stdin();
    if (lines.length < 2) {
      console.error(`design problems need two input lines, got ${lines.length}`);
      process.exit(1);
    }
    const ops = JSON.parse(lines[0]);
    const rawArgs = JSON.parse(lines[1]);
    if (ops.length !== rawArgs.length) {
      console.error(`got ${ops.length} operations but ${rawArgs.length} argument lists`);
      process.exit(1);
    }

    const results = [];
    let instance = null;

    for (let i = 0; i < ops.length; i++) {
      const types = typesByMethod[ops[i]] || [];
      // Re-encode: parse works on the textual form, and these arrived already
      // decoded as part of the outer array.
      const args = rawArgs[i].map((raw, j) =>
        types[j] ? __leetui.parse(types[j], JSON.stringify(raw)) : raw);

      if (i === 0) {
        instance = new ctor(...args);
        results.push(null);
        continue;
      }
      const out = instance[ops[i]](...args);
      results.push(out === undefined ? null : out);
    }
    console.log("[" + results.map(__leetui.dump).join(",") + "]");
  },
};
