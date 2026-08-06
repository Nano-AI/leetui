package leetcode

// GraphQL documents.
//
// These mirror what leetcode.com's own frontend sends. They were verified against the
// live endpoint on 2026-08-06 (4,013 problems returned).
//
// If a query starts failing, LeetCode changed their schema. Diff against a real browser
// request (devtools → Network → graphql → Payload) before rewriting anything — the
// field set is usually what moved, not the operation name.

const qGlobalData = `
query globalData {
  userStatus {
    userId
    isSignedIn
    isPremium
    username
  }
}`

const qProblemList = `
query problemsetQuestionList($categorySlug: String, $limit: Int, $skip: Int, $filters: QuestionListFilterInput) {
  problemsetQuestionList: questionList(
    categorySlug: $categorySlug
    limit: $limit
    skip: $skip
    filters: $filters
  ) {
    total: totalNum
    questions: data {
      acRate
      difficulty
      frontendQuestionId: questionFrontendId
      paidOnly: isPaidOnly
      status
      title
      titleSlug
      topicTags { name slug }
      hasSolution
      hasVideoSolution
    }
  }
}`

const qQuestionData = `
query questionData($titleSlug: String!) {
  question(titleSlug: $titleSlug) {
    questionId
    questionFrontendId
    title
    titleSlug
    content
    isPaidOnly
    difficulty
    likes
    dislikes
    topicTags { name slug }
    codeSnippets { lang langSlug code }
    hints
    sampleTestCase
    exampleTestcases
    metaData
    enableRunCode
    hasSolution
  }
}`
