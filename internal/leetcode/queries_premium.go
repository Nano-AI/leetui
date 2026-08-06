package leetcode

// Premium GraphQL documents (D-006).
//
// Verified against the live endpoint on 2026-08-06. Two of these answer without a
// session, which is worth knowing before assuming a failure is a premium gate:
//
//	companyTags       984 companies, name + slug + questionCount, SIGNED OUT
//	favoriteDetailV2  a company pack's size, SIGNED OUT
//	favoriteQuestionList  the pack's problems — PREMIUM, empty list when gated
//	question.solution     the editorial — free problems are public, premium ones gate
//	                      on canSeeDetail rather than erroring
//
// The gated ones return an EMPTY RESULT rather than an error, so callers must check for
// emptiness explicitly; see ErrPremiumRequired's use in company.go and editorial.go.

const qCompanyTags = `
query companyTags {
  companyTags {
    name
    slug
    questionCount
  }
}`

// qFavoriteDetail reads a pack's metadata. It is also the only way to tell a real
// company/timeframe pair from a typo: an unknown favoriteSlug returns null rather than
// an error, and paging an unknown slug would otherwise look like an empty pack.
const qFavoriteDetail = `
query favoriteDetailV2($favoriteSlug: String!) {
  favoriteDetailV2(favoriteSlug: $favoriteSlug) {
    name
    slug
    questionNumber
  }
}`

// qFavoriteQuestionList pages a company pack.
//
// frequency is how often the company has asked it, on LeetCode's own scale. It is
// relative within a pack and is not comparable across companies.
const qFavoriteQuestionList = `
query favoriteQuestionList($favoriteSlug: String!, $limit: Int, $skip: Int, $version: String) {
  favoriteQuestionList(
    favoriteSlug: $favoriteSlug
    limit: $limit
    skip: $skip
    version: $version
  ) {
    totalLength
    hasMore
    questions {
      titleSlug
      title
      questionFrontendId
      difficulty
      status
      paidOnly
      frequency
      acRate
      topicTags { name slug }
    }
  }
}`

// qSolution reads the official editorial.
//
// canSeeDetail is the field that matters: a premium-gated editorial still returns a node
// with a title and paidOnly set, so the pane can name what it is withholding instead of
// showing a blank or an error (D-006).
const qSolution = `
query questionSolution($titleSlug: String!) {
  question(titleSlug: $titleSlug) {
    solution {
      id
      title
      content
      paidOnly
      canSeeDetail
      hasVideoSolution
    }
  }
}`
