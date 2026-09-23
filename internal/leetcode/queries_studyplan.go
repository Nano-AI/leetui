package leetcode

// Study plan GraphQL documents (D-031).
//
// Verified against the live endpoint on 2026-08-09. Every one of these answers WITHOUT a
// session, which is the whole reason study plans are worth having next to company packs:
//
//	studyPlanV2Detail   a plan and all of its questions, SIGNED OUT, one request
//	studyPlansV2ByTag   the plans carrying one tag, SIGNED OUT
//	studyPlanV2Catalogs the five category headings, SIGNED OUT
//
// A premium plan gates the same way a company pack does — planSubGroups comes back EMPTY
// rather than erroring, with premiumOnly set to say why. Callers must test for emptiness;
// see ErrPremiumRequired's use in studyplan.go.
//
// studyPlansV2ByUpc exists too and is the website's "Ongoing" row, but it answers
// "User is not authenticated" signed out and is not used here: progress is already
// tracked locally against the problems table, which works for a free account.

// qStudyPlanDetail reads a whole plan in one request.
//
// There is no limit/skip on this field and none is needed: Top Interview 150 returns all
// 150 questions and its 23 subgroups in a single response. That is the difference between
// a plan and a company pack, which pages at 100 and needs 24 requests for Google.
const qStudyPlanDetail = `
query studyPlanV2Detail($planSlug: String!) {
  studyPlanV2Detail(planSlug: $planSlug) {
    slug
    name
    highlight
    questionNum
    premiumOnly
    planSubGroups {
      slug
      name
      questionNum
      questions {
        titleSlug
        title
        questionFrontendId
        difficulty
        status
        paidOnly
        topicTags { name slug }
      }
    }
  }
}`

// qStudyPlansByTag lists the plans under one tag.
//
// The tag vocabulary is NOT LeetCode's topic tags. It is a short curated set — see
// DiscoveryTags — and an unknown tag returns an empty list rather than an error, which is
// indistinguishable from a tag that genuinely has no plans. That is why discovery is a
// union with SeedPlans rather than a sweep on its own.
const qStudyPlansByTag = `
query studyPlansV2ByTag($tagSlug: String!, $limit: Int) {
  studyPlansV2ByTag(tagSlug: $tagSlug, limit: $limit) {
    total
    hasMore
    studyPlans {
      slug
      name
      highlight
      questionNum
      premiumOnly
    }
  }
}`
