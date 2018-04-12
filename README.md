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

# Dependencies

The bot needs the following binaries in $PATH:

- go
- git
- megacheck (which also requires gcc, apparently)


## When to comment

We should comment only on new code.
But what's new code?

- avoid vendored and code automatically generated
- avoid old code, only comment on code that has been modified/created

But what about if you modify a line of code and that causes a problem in a different line?

Example:

```go
var a int
fmt.Printf("%d", a)
```

If we change to 

```go
var a string
fmt.Printf("%d", a)
```

We'll cause a warning on the Printf because %d is being passed a string, but that line didn't change!