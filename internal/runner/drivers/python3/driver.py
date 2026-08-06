"""Local test driver for LeetCode Python solutions.

Adapted from leetgo (https://github.com/j178/leetgo), MIT licensed,
Copyright (c) 2022 Jo. Rewritten to be a single self-contained file so it needs
no `pip install` step — leetui writes it next to the solution and runs it.

leetui generates a small entry file that imports this and calls `run`. The
serialization format matches LeetCode's own: JSON-ish literals, one per line
for each parameter, with `[]` for an empty list or tree node.

WHAT THIS CANNOT KNOW
---------------------
`metaData` gives parameter names, types, and the return type. It does NOT say
whether a problem mutates its input in place, accepts answers in any order, or
tolerates floating-point error. Those are handled by the comparator overrides
in Go, not here, and any disagreement between this driver and the judge is
reported as "verify remotely" rather than as a failure (D-003).
"""

import json
import sys
from typing import Any, List, Optional


class ListNode:
    """LeetCode's singly-linked list node."""

    def __init__(self, val: int = 0, next: "Optional[ListNode]" = None):
        self.val = val
        self.next = next

    @staticmethod
    def build(values: List[int]) -> "Optional[ListNode]":
        head = None
        for v in reversed(values):
            head = ListNode(v, head)
        return head

    def serialize(self) -> str:
        out, node, seen = [], self, set()
        while node is not None:
            # A cycle would otherwise hang the runner rather than fail it. Some
            # problems build one deliberately (linked list cycle II).
            if id(node) in seen:
                out.append("...cycle")
                break
            seen.add(id(node))
            out.append(json.dumps(node.val))
            node = node.next
        return "[" + ",".join(out) + "]"


class TreeNode:
    """LeetCode's binary tree node."""

    def __init__(self, val: int = 0, left: "Optional[TreeNode]" = None,
                 right: "Optional[TreeNode]" = None):
        self.val = val
        self.left = left
        self.right = right

    @staticmethod
    def build(values: List[Any]) -> "Optional[TreeNode]":
        """Build from LeetCode's level-order form, where null marks a gap."""
        if not values or values[0] is None:
            return None
        root = TreeNode(values[0])
        queue, i = [root], 1
        while queue and i < len(values):
            node = queue.pop(0)
            if i < len(values):
                if values[i] is not None:
                    node.left = TreeNode(values[i])
                    queue.append(node.left)
                i += 1
            if i < len(values):
                if values[i] is not None:
                    node.right = TreeNode(values[i])
                    queue.append(node.right)
                i += 1
        return root

    def serialize(self) -> str:
        """Level-order with trailing nulls trimmed, matching LeetCode's output."""
        out, queue = [], [self]
        while queue:
            node = queue.pop(0)
            if node is None:
                out.append("null")
                continue
            out.append(json.dumps(node.val))
            queue.append(node.left)
            queue.append(node.right)
        while out and out[-1] == "null":
            out.pop()
        return "[" + ",".join(out) + "]"


def deserialize(type_name: str, raw: str) -> Any:
    """Turn one line of test input into a Python value.

    type_name is LeetCode's own spelling from metaData: "integer",
    "integer[]", "string", "ListNode", "TreeNode", "character", and so on.
    """
    raw = raw.strip()
    if type_name == "ListNode":
        return ListNode.build(json.loads(raw))
    if type_name == "TreeNode":
        return TreeNode.build(json.loads(raw))
    if type_name in ("ListNode[]", "TreeNode[]"):
        builder = ListNode.build if type_name.startswith("List") else TreeNode.build
        return [builder(v) for v in json.loads(raw)]
    # Everything else is plain JSON: integers, floats, strings, booleans, and
    # arrays of them. "character" arrives as a one-character JSON string.
    return json.loads(raw)


def serialize(value: Any) -> str:
    """Render a return value the way LeetCode prints it."""
    if isinstance(value, (ListNode, TreeNode)):
        return value.serialize()
    if isinstance(value, list):
        return "[" + ",".join(serialize(v) for v in value) + "]"
    if value is None:
        return "null"
    # json.dumps gives true/false rather than True/False, which is what the
    # judge prints.
    return json.dumps(value)


def run(solution_cls, method: str, param_types: List[str],
        mutates: Optional[str] = None) -> None:
    """Read one test case from stdin, run it, print the result.

    Input is one line per parameter. Output is a single line.

    mutates names the parameter to print instead of the return value, for
    problems whose answer is the modified input (remove duplicates, sort
    colors). leetui passes it from its override table.
    """
    lines = sys.stdin.read().splitlines()
    if len(lines) < len(param_types):
        raise SystemExit(
            "expected %d input lines, got %d" % (len(param_types), len(lines)))

    args = [deserialize(t, lines[i]) for i, t in enumerate(param_types)]
    result = getattr(solution_cls(), method)(*args)

    if mutates is not None:
        index = int(mutates)
        # In-place problems return the new length and expect the prefix.
        if isinstance(result, int) and isinstance(args[index], list):
            print(serialize(args[index][:result]))
        else:
            print(serialize(args[index]))
        return

    print(serialize(result))


def run_design(cls, param_types_by_method) -> None:
    """Run a design problem: a class, then a sequence of operations.

    LeetCode sends these as two lines. The first names the operations, starting
    with the constructor; the second gives each one's arguments:

        ["LRUCache","put","put","get"]
        [[2],[1,1],[2,2],[1]]

    Output is one list holding each call's return value, with null for the
    constructor and for methods that return nothing:

        [null,null,null,1]

    param_types_by_method maps a method name to its parameter types, so
    arguments are deserialized the same way as for a plain function — a
    TreeNode argument to a method still has to become a TreeNode.
    """
    lines = sys.stdin.read().splitlines()
    if len(lines) < 2:
        raise SystemExit("design problems need two input lines, got %d" % len(lines))

    ops = json.loads(lines[0])
    raw_args = json.loads(lines[1])
    if len(ops) != len(raw_args):
        raise SystemExit("got %d operations but %d argument lists" % (len(ops), len(raw_args)))

    results: List[Any] = []
    instance = None

    for i, op in enumerate(ops):
        types = param_types_by_method.get(op, [])
        args = []
        for j, raw in enumerate(raw_args[i]):
            ty = types[j] if j < len(types) else None
            # Re-encode: deserialize works on the textual form, and these arrived
            # already parsed as part of the outer array.
            args.append(deserialize(ty, json.dumps(raw)) if ty else raw)

        if i == 0:
            # The first operation is always the constructor.
            instance = cls(*args)
            results.append(None)
            continue

        results.append(getattr(instance, op)(*args))

    print("[" + ",".join(serialize(r) for r in results) + "]")
