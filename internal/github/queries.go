package github

const prFilesQuery = `
query($owner: String!, $repo: String!, $number: Int!, $after: String) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $number) {
      id
      title
      additions
      deletions
      changedFiles
      files(first: 100, after: $after) {
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
          path
          additions
          deletions
          viewerViewedState
        }
      }
    }
  }
}
`

const markFileAsViewedMutation = `
mutation($pullRequestId: ID!, $path: String!) {
  markFileAsViewed(input: {pullRequestId: $pullRequestId, path: $path}) {
    pullRequest {
      id
    }
  }
}
`

const unmarkFileAsViewedMutation = `
mutation($pullRequestId: ID!, $path: String!) {
  unmarkFileAsViewed(input: {pullRequestId: $pullRequestId, path: $path}) {
    pullRequest {
      id
    }
  }
}
`

const reviewThreadsQuery = `
query($owner: String!, $repo: String!, $number: Int!, $after: String) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $number) {
      reviewThreads(first: 100, after: $after) {
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
          id
          isResolved
          path
          line
          diffSide
          comments(first: 100) {
            nodes {
              id
              body
              author {
                login
              }
              createdAt
            }
          }
        }
      }
    }
  }
}
`

const addReviewCommentMutation = `
mutation($pullRequestId: ID!, $body: String!, $path: String!, $line: Int!, $side: DiffSide!) {
  addPullRequestReview(input: {
    pullRequestId: $pullRequestId,
    event: COMMENT,
    threads: [{
      body: $body,
      path: $path,
      line: $line,
      side: $side
    }]
  }) {
    pullRequestReview {
      id
    }
  }
}
`

const replyToThreadMutation = `
mutation($threadId: ID!, $body: String!) {
  addPullRequestReviewThreadReply(input: {
    pullRequestReviewThreadId: $threadId,
    body: $body
  }) {
    comment {
      id
    }
  }
}
`

const resolveThreadMutation = `
mutation($threadId: ID!) {
  resolveReviewThread(input: {threadId: $threadId}) {
    thread {
      id
      isResolved
    }
  }
}
`

const unresolveThreadMutation = `
mutation($threadId: ID!) {
  unresolveReviewThread(input: {threadId: $threadId}) {
    thread {
      id
      isResolved
    }
  }
}
`

const openPRsQuery = `
query($owner: String!, $repo: String!, $after: String) {
  repository(owner: $owner, name: $repo) {
    pullRequests(first: 100, after: $after, states: OPEN, orderBy: {field: UPDATED_AT, direction: DESC}) {
      pageInfo {
        hasNextPage
        endCursor
      }
      nodes {
        number
        title
        isDraft
        updatedAt
        author {
          login
        }
      }
    }
  }
}
`

const submitReviewMutation = `
mutation($pullRequestId: ID!, $event: PullRequestReviewEvent!, $body: String) {
  addPullRequestReview(input: {
    pullRequestId: $pullRequestId,
    event: $event,
    body: $body
  }) {
    pullRequestReview {
      id
    }
  }
}
`
