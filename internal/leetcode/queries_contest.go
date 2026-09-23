package leetcode

// Contest GraphQL documents (D-036).
//
// Verified against the live endpoint on 2026-08-30. All three answer WITHOUT a session,
// which is what makes a contest board worth having: a free, signed-out account can see
// the schedule and, once a contest opens, the four problems in it.
//
//	upcomingContests    the contests that have not started, SIGNED OUT
//	contest             one contest's title, start, duration, SIGNED OUT
//	contestQuestionList one contest's questions, SIGNED OUT once the contest opens
//
// The one field that needs a session is isAc, which is the signed-in user's own progress.
// Signed out it comes back false for every question, which reads as "nothing solved" and
// is the truth for an anonymous caller.
//
// allContests exists and returns EVERY contest ever run, newest first, in one response —
// about 700 entries and growing. It is not used. upcomingContests is the half that
// matters and it is three orders of magnitude smaller; a past contest is reached by slug.

// qUpcomingContests lists the contests that have not started yet.
//
// Verified to return only contests whose registration is open, which is not the same as
// "every contest with a start time in the future": allContests knows about two or three
// beyond this list. That is deliberate on LeetCode's side and it is the right list to
// show, since a contest you cannot register for is not yet a thing you can plan around.
const qUpcomingContests = `
query upcomingContests {
  upcomingContests {
    title
    titleSlug
    startTime
    duration
  }
}`

// qContest reads one contest's frame — everything except its questions.
//
// startTime is a Unix second, NOT a millisecond, and duration is in seconds. A weekly is
// 5400 (90 minutes) and a biweekly is the same. Both are read rather than assumed: the
// countdown is the feature, and a hardcoded 90 minutes would be silently wrong the first
// time LeetCode runs a contest of another length.
const qContest = `
query contest($titleSlug: String!) {
  contest(titleSlug: $titleSlug) {
    title
    titleSlug
    startTime
    originStartTime
    duration
    isVirtual
    containsPremium
  }
}`

// qContestQuestion reads ONE contest problem's full statement.
//
// THIS IS THE ONLY WAY TO READ A LIVE CONTEST'S PROBLEMS, and finding that out cost a
// contest. While a contest runs its problems are not in the problem set, and the ordinary
// `question(titleSlug:)` returns **null** for every one of them — not an error, not a
// permission failure, just null, with HTTP 200. Referer makes no difference and neither
// does being registered. Verified against Weekly Contest 517 while it was live.
//
// `contestQuestion(contestSlug:questionSlug:)` serves them, and the wrapper is the point:
// the outer `ContestQuestionDetailNode` carries contest bookkeeping, and `question` is a
// `ContestQuestionNodeV2` holding the real statement, metaData and snippets — everything
// a local run needs.
//
// NEEDS A SESSION. Signed out it answers "user is not authenticated", which is the one
// place in this feature where signing in is not optional.
//
// Two fields are spelled differently here than on the ordinary question type, and using
// the familiar names fails the whole query:
//
//	exampleTestcaseList   a LIST, where questionData has exampleTestcases as one string
//	(no topicTags)        the V2 node has none, so a contest problem carries no tags
//	(no isPaidOnly)       likewise; a contest problem is never premium-gated
const qContestQuestion = `
query contestQuestion($contestSlug: String!, $questionSlug: String!) {
  contestQuestion(contestSlug: $contestSlug, questionSlug: $questionSlug) {
    question {
      questionId
      questionFrontendId
      title
      titleSlug
      content
      difficulty
      metaData
      enableRunCode
      exampleTestcaseList
      codeSnippets {
        lang
        langSlug
        code
      }
    }
  }
}`

// qContestQuestions reads the questions in one contest.
//
// EMPTY IS THE NORMAL ANSWER BEFORE THE START. LeetCode does not error on a contest that
// has not opened, it returns an empty list — the same shape as a slug that does not
// exist. Callers must not treat empty as failure; see Contest's Open() for the
// distinction the UI actually needs.
//
// credit is the contest's own scoring weight (3/4/5/6 for a weekly's four problems), not
// a difficulty. It is the closest thing the contest API gives to one, and it is what the
// board sorts on, because the running order of a contest IS its running order.
const qContestQuestions = `
query contestQuestionList($contestSlug: String!) {
  contestQuestionList(contestSlug: $contestSlug) {
    questionId
    title
    titleSlug
    credit
    isAc
  }
}`
