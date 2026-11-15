# Frozen interface projection costs

This fixture compares one direct Frozen interface call with the same call
through its generated local projection. The implementation returns one integer or a tuple with an integer, found flag,
caller buffer, and error.
The projection stores the same two-word interface header. The test checks zero
allocations for a Cell load, interface field projection, and method call, including the tuple result.

Run the focused measurements from the repository root:

```sh
go test ./internal/projectiongen/testdata/interfacecost -run '^$' -bench 'Interface$' -benchmem -count=3
```

These measurements cover dispatch and the generated adapter. A user's method
body and Go interface conversion have their own ordinary costs. Record measured
timings in the PR, not in the public guide.
