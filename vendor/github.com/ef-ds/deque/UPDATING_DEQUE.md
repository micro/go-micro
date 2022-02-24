## Updating Deque and Checking the Results

If you want to make changes to deque and run the tests to check the effect on performance and memory,
we suggest you run all the benchmark tests locally once using below command.

```
go test -benchmem -count 10 -timeout 60m -bench="Deque*" -run=^$ > testdata/deque.txt
```

Then make the changes and re-run the tests using below command (notice the output file now is deque2.txt).

```
go test -benchmem -count 10 -timeout 60m -bench="Deque*" -run=^$ > testdata/deque2.txt
```

Then run the [test-splitter](https://github.com/ef-ds/tools/tree/master/testsplitter) tool as follow:

```
go run *.go --file PATH_TO_TESTDATA/deque2.txt --suffix 2
```

Test-splitter should create each file with the "2" suffix, so now we have the test file for both, the old and this new
test run. Use below commands to test the effect of the changes for each test suite.

```
benchstat testdata/BenchmarkMicroserviceQueue.txt testdata/BenchmarkMicroserviceQueue2.txt
benchstat testdata/BenchmarkMicroserviceStack.txt testdata/BenchmarkMicroserviceStack2.txt
benchstat testdata/BenchmarkFillQueue.txt testdata/BenchmarkFillQueue2.txt
benchstat testdata/BenchmarkFillStack.txt testdata/BenchmarkFillStack2.txt
benchstat testdata/BenchmarkRefillQueue.txt testdata/BenchmarkRefillQueue2.txt
benchstat testdata/BenchmarkRefillStack.txt testdata/BenchmarkRefillStack2.txt
benchstat testdata/BenchmarkRefillFullQueue.txt testdata/BenchmarkRefillFullQueue2.txt
benchstat testdata/BenchmarkRefillFullStack.txt testdata/BenchmarkRefillFullStack2.txt
benchstat testdata/BenchmarkSlowIncreaseQueue.txt testdata/BenchmarkSlowIncreaseQueue2.txt
benchstat testdata/BenchmarkSlowIncreaseStack.txt testdata/BenchmarkSlowIncreaseStack2.txt
benchstat testdata/BenchmarkSlowDecreaseQueue.txt testdata/BenchmarkSlowDecreaseQueue2.txt
benchstat testdata/BenchmarkSlowIncreaseStack.txt testdata/BenchmarkSlowIncreaseStack2.txt
benchstat testdata/BenchmarkStableQueue.txt testdata/BenchmarkStableQueue2.txt
benchstat testdata/BenchmarkStableStack.txt testdata/BenchmarkStableStack2.txt
```
