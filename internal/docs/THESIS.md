# Go Micro direction

Go Micro is a framework for services and agents. Services expose typed operations
through RPC and discovery. Agents use these operations as tools and can delegate
to other agents. The CLI and gateways provide access to the same components.

Keep the framework useful independently of any product. Studio provides a
developer workspace; Mu provides an application runtime. Neither should impose
product-specific behaviour on the framework.

The standalone flow abstraction has been removed. Focus on simple service and
agent APIs, dependable tool execution, cancellation, checkpointing and inspection.
Prove changes with real service-to-agent journeys rather than adding abstractions.
