// Mirrored by hand from apps/api/internal/constants/moemoepoint.go, which is the
// only place the server reads. The /point page quotes every one of these numbers
// at users, and a page that quotes the wrong price is worse than no page —
// apps/api/internal/constants/frontend_mirror_test.go fails the moment the two
// copies disagree.
export const KUN_MOEMOEPOINT = {
  createTopic: 3,
  createGalgame: 3,
  createResource: 3,
  createToolset: 3,
  reply: 1,
  prMerge: 1,
  consumeSection: 10,
  upvoteSender: 10,
  upvoteOwner: 5,
  ratingHigh: 10,
  ratingMedium: 5,
  ratingLow: 3,
  ratingLenHigh: 666,
  ratingLenMedium: 233,
  quizCreate: 2,
  bestAnswer: 7,
  checkinMax: 7,
  dailyTopicPerPoint: 10
} as const
