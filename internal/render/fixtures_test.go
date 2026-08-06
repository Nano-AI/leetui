package render

// realStatement is the shape LeetCode actually serves: pretty-printed HTML, <pre>
// examples, <sup> in constraints, and an <img> diagram.
const realStatement = `<p>Given an array of integers <code>nums</code>&nbsp;and an integer <code>target</code>, return <em>indices of the two numbers such that they add up to <code>target</code></em>.</p>

<p>You may assume that each input would have <strong>exactly one solution</strong>, and you may not use the <em>same</em> element twice.</p>

<img alt="dp state table" src="https://assets.leetcode.com/uploads/2021/01/example.png" />

<p>&nbsp;</p>
<p><strong class="example">Example 1:</strong></p>

<pre>
<strong>Input:</strong> nums = [2,7,11,15], target = 9
<strong>Output:</strong> [0,1]
<strong>Explanation:</strong> Because nums[0] + nums[1] == 9, we return [0, 1].
</pre>

<p><strong>Constraints:</strong></p>

<ul>
	<li><code>2 &lt;= nums.length &lt;= 10<sup>4</sup></code></li>
	<li><code>-10<sup>9</sup> &lt;= nums[i] &lt;= 10<sup>9</sup></code></li>
</ul>

<p><strong>Follow-up:</strong> Can you come up with an algorithm that is less than <code>O(n<sup>2</sup>)</code> time complexity?</p>`
