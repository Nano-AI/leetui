// Local test driver for LeetCode C++ solutions.
//
// Adapted from leetgo (https://github.com/j178/leetgo), MIT licensed,
// Copyright (c) 2022 Jo. Rewritten as a single header so no build system or
// package manager is needed — leetui writes it beside the solution and compiles
// the two together.
//
// C++ has no standard JSON, so this carries a small recursive-descent parser for
// exactly the shapes LeetCode sends: numbers, strings, booleans, null, and
// nested arrays. It is not a general JSON library and should not become one.
#pragma once

// LeetCode's judge compiles C++ submissions with <bits/stdc++.h> and
// `using namespace std;` already in scope, so its starter snippets say
// `vector<int>` rather than `std::vector<int>`. libc++ on macOS has no
// bits/stdc++.h, so the common headers are listed explicitly and the namespace
// is opened at the bottom of this file — after leetui's own declarations, so
// nothing here depends on it.
#include <algorithm>
#include <array>
#include <bitset>
#include <climits>
#include <cmath>
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <deque>
#include <functional>
#include <iostream>
#include <iterator>
#include <limits>
#include <list>
#include <map>
#include <memory>
#include <numeric>
#include <queue>
#include <set>
#include <sstream>
#include <stack>
#include <string>
#include <tuple>
#include <unordered_map>
#include <unordered_set>
#include <utility>
#include <vector>

namespace leetui {

// ---------------------------------------------------------------------------
// A minimal JSON value
// ---------------------------------------------------------------------------

struct Json {
    enum Kind { Null, Bool, Number, String, Array } kind = Null;
    bool boolean = false;
    double number = 0;
    std::string str;
    std::vector<Json> items;

    bool isNull() const { return kind == Null; }
    long long asInt() const { return static_cast<long long>(number); }
};

class Parser {
   public:
    explicit Parser(const std::string& text) : s_(text) {}

    Json parse() {
        skip();
        Json v = value();
        return v;
    }

   private:
    const std::string& s_;
    size_t i_ = 0;

    void skip() {
        while (i_ < s_.size() && std::isspace(static_cast<unsigned char>(s_[i_]))) i_++;
    }

    [[noreturn]] void fail(const std::string& why) {
        std::cerr << "parse input: " << why << " at offset " << i_ << "\n";
        std::exit(1);
    }

    Json value() {
        skip();
        if (i_ >= s_.size()) fail("unexpected end of input");

        switch (s_[i_]) {
            case '[': return array();
            case '"': return string();
            case 't': case 'f': return boolean();
            case 'n': return null();
            default: return number();
        }
    }

    Json array() {
        Json v;
        v.kind = Json::Array;
        i_++;  // [
        skip();
        if (i_ < s_.size() && s_[i_] == ']') { i_++; return v; }

        while (i_ < s_.size()) {
            v.items.push_back(value());
            skip();
            if (i_ < s_.size() && s_[i_] == ',') { i_++; continue; }
            if (i_ < s_.size() && s_[i_] == ']') { i_++; return v; }
            fail("expected ',' or ']'");
        }
        fail("unterminated array");
    }

    Json string() {
        Json v;
        v.kind = Json::String;
        i_++;  // opening quote
        while (i_ < s_.size() && s_[i_] != '"') {
            if (s_[i_] == '\\' && i_ + 1 < s_.size()) {
                i_++;
                switch (s_[i_]) {
                    case 'n': v.str += '\n'; break;
                    case 't': v.str += '\t'; break;
                    case 'r': v.str += '\r'; break;
                    default: v.str += s_[i_];
                }
            } else {
                v.str += s_[i_];
            }
            i_++;
        }
        if (i_ >= s_.size()) fail("unterminated string");
        i_++;  // closing quote
        return v;
    }

    Json boolean() {
        Json v;
        v.kind = Json::Bool;
        if (s_.compare(i_, 4, "true") == 0) { v.boolean = true; i_ += 4; return v; }
        if (s_.compare(i_, 5, "false") == 0) { v.boolean = false; i_ += 5; return v; }
        fail("expected true or false");
    }

    Json null() {
        if (s_.compare(i_, 4, "null") != 0) fail("expected null");
        i_ += 4;
        return Json{};
    }

    Json number() {
        size_t start = i_;
        while (i_ < s_.size() &&
               (std::isdigit(static_cast<unsigned char>(s_[i_])) || s_[i_] == '-' ||
                s_[i_] == '+' || s_[i_] == '.' || s_[i_] == 'e' || s_[i_] == 'E')) {
            i_++;
        }
        if (start == i_) fail("expected a number");
        Json v;
        v.kind = Json::Number;
        v.number = std::stod(s_.substr(start, i_ - start));
        return v;
    }
};

inline Json parse(const std::string& text) { return Parser(text).parse(); }

}  // namespace leetui

// ---------------------------------------------------------------------------
// LeetCode's node types
// ---------------------------------------------------------------------------
//
// Declared here rather than in the solution file so the starter snippet stays
// exactly as LeetCode wrote it — its own definitions arrive commented out.

struct ListNode {
    int val;
    ListNode* next;
    ListNode() : val(0), next(nullptr) {}
    ListNode(int x) : val(x), next(nullptr) {}
    ListNode(int x, ListNode* n) : val(x), next(n) {}
};

struct TreeNode {
    int val;
    TreeNode* left;
    TreeNode* right;
    TreeNode() : val(0), left(nullptr), right(nullptr) {}
    TreeNode(int x) : val(x), left(nullptr), right(nullptr) {}
    TreeNode(int x, TreeNode* l, TreeNode* r) : val(x), left(l), right(r) {}
};

namespace leetui {

// --- building arguments ----------------------------------------------------

inline int toInt(const Json& j) { return static_cast<int>(j.number); }
inline long long toLong(const Json& j) { return j.asInt(); }
inline double toDouble(const Json& j) { return j.number; }
inline bool toBool(const Json& j) { return j.boolean; }
inline std::string toString(const Json& j) { return j.str; }
// LeetCode sends a character as a one-character string.
inline char toChar(const Json& j) { return j.str.empty() ? '\0' : j.str[0]; }

template <typename T, typename F>
std::vector<T> toVector(const Json& j, F convert) {
    std::vector<T> out;
    out.reserve(j.items.size());
    for (const auto& item : j.items) out.push_back(convert(item));
    return out;
}

inline std::vector<int> toIntVector(const Json& j) { return toVector<int>(j, toInt); }
inline std::vector<double> toDoubleVector(const Json& j) { return toVector<double>(j, toDouble); }
inline std::vector<bool> toBoolVector(const Json& j) { return toVector<bool>(j, toBool); }
inline std::vector<std::string> toStringVector(const Json& j) { return toVector<std::string>(j, toString); }
inline std::vector<char> toCharVector(const Json& j) { return toVector<char>(j, toChar); }

inline std::vector<std::vector<int>> toIntGrid(const Json& j) {
    return toVector<std::vector<int>>(j, toIntVector);
}
inline std::vector<std::vector<char>> toCharGrid(const Json& j) {
    return toVector<std::vector<char>>(j, toCharVector);
}
inline std::vector<std::vector<std::string>> toStringGrid(const Json& j) {
    return toVector<std::vector<std::string>>(j, toStringVector);
}

inline ListNode* toList(const Json& j) {
    ListNode* head = nullptr;
    for (auto it = j.items.rbegin(); it != j.items.rend(); ++it) {
        head = new ListNode(toInt(*it), head);
    }
    return head;
}

// toTree reads LeetCode's level-order form, where null marks a gap.
inline TreeNode* toTree(const Json& j) {
    if (j.items.empty() || j.items[0].isNull()) return nullptr;

    TreeNode* root = new TreeNode(toInt(j.items[0]));
    std::queue<TreeNode*> q;
    q.push(root);
    size_t i = 1;

    while (!q.empty() && i < j.items.size()) {
        TreeNode* node = q.front();
        q.pop();
        if (i < j.items.size()) {
            if (!j.items[i].isNull()) {
                node->left = new TreeNode(toInt(j.items[i]));
                q.push(node->left);
            }
            i++;
        }
        if (i < j.items.size()) {
            if (!j.items[i].isNull()) {
                node->right = new TreeNode(toInt(j.items[i]));
                q.push(node->right);
            }
            i++;
        }
    }
    return root;
}

// --- printing results ------------------------------------------------------

inline std::string dump(int v) { return std::to_string(v); }
inline std::string dump(long long v) { return std::to_string(v); }
inline std::string dump(bool v) { return v ? "true" : "false"; }
inline std::string dump(char v) { return std::string("\"") + v + "\""; }
inline std::string dump(const std::string& v) { return "\"" + v + "\""; }

inline std::string dump(double v) {
    std::ostringstream os;
    os.precision(5);
    os << std::fixed << v;
    return os.str();
}

inline std::string dump(ListNode* n) {
    std::string out = "[";
    std::set<ListNode*> seen;
    bool first = true;
    while (n != nullptr) {
        // A cycle would hang the runner rather than fail it; some problems build
        // one deliberately.
        if (seen.count(n)) { out += first ? "\"...cycle\"" : ",\"...cycle\""; break; }
        seen.insert(n);
        if (!first) out += ",";
        out += std::to_string(n->val);
        first = false;
        n = n->next;
    }
    return out + "]";
}

// dump(TreeNode*) emits level-order with trailing nulls trimmed, as the judge does.
inline std::string dump(TreeNode* root) {
    if (root == nullptr) return "[]";

    std::vector<std::string> parts;
    std::queue<TreeNode*> q;
    q.push(root);
    while (!q.empty()) {
        TreeNode* node = q.front();
        q.pop();
        if (node == nullptr) { parts.push_back("null"); continue; }
        parts.push_back(std::to_string(node->val));
        q.push(node->left);
        q.push(node->right);
    }
    while (!parts.empty() && parts.back() == "null") parts.pop_back();

    std::string out = "[";
    for (size_t i = 0; i < parts.size(); i++) {
        if (i) out += ",";
        out += parts[i];
    }
    return out + "]";
}

template <typename T>
std::string dump(const std::vector<T>& v) {
    std::string out = "[";
    for (size_t i = 0; i < v.size(); i++) {
        if (i) out += ",";
        out += dump(v[i]);
    }
    return out + "]";
}

// std::vector<bool> is a bitset proxy, so its elements do not bind to dump(bool).
inline std::string dump(const std::vector<bool>& v) {
    std::string out = "[";
    for (size_t i = 0; i < v.size(); i++) {
        if (i) out += ",";
        out += (v[i] ? "true" : "false");
    }
    return out + "]";
}

// prefix trims an in-place answer to the length the solution reported.
template <typename T>
std::vector<T> prefix(const std::vector<T>& v, int n) {
    if (n < 0 || static_cast<size_t>(n) > v.size()) return v;
    return std::vector<T>(v.begin(), v.begin() + n);
}

// readLines takes one line per parameter from stdin.
inline std::vector<std::string> readLines(size_t n) {
    std::vector<std::string> lines;
    std::string line;
    while (std::getline(std::cin, line)) lines.push_back(line);
    if (lines.size() < n) {
        std::cerr << "expected " << n << " input lines, got " << lines.size() << "\n";
        std::exit(1);
    }
    lines.resize(n);
    return lines;
}

}  // namespace leetui

// Opened last, and deliberately: LeetCode's snippets are written as if the judge
// had already done this, and they will not compile without it. Everything above
// is fully qualified so this cannot change the meaning of the driver itself.
using namespace std;
