# Runtime

A runtime for self governing services.

## Overview

In recent years we've started to develop complex architectures for the pipeline between writing code and running it. This 
philosophy of build, run, manage or however many variations, has created a number of layers of abstraction that make it 
all the more difficult to run code.

Runtime manages the lifecycle of a service from source to running process. If the source is the *source of truth* then 
everything in between running is wasted breath. Applications should be self governing and self sustaining. 
To enable that we need libraries which make it possible.

Runtime will fetch source code, build a binary and execute it. Any Go program that uses this library should be able 
to run dependencies or itself with ease, with the ability to update itself as the source is updated.

## Features

- **Source** - Fetches source whether it be git, go, docker, etc
- **Package** - Compiles the source into a binary which can be executed
- **Process** - Executes a binary and creates a running process

## Usage

TODO


