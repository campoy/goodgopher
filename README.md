# Good Gopher: Go Code Reviews as a Service

Good Gopher is a GitHub bot that reviews Pull Requests and gives
advice on how to improve the code in it.

Current status: experimental

## Design

1. receive GitHub webhook about a new pull request
2. fetch code related to the PR
3. run analysis tools (maybe concurrently?)
    1. Maybe filter out comments on code that has not changed?
4. report all output as comments in the PR
