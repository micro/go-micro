package kubernetes

// CRDManifests contains the alpha CRDs for Go Micro lifecycle resources.
var CRDManifests = map[Kind]string{
	KindAgent:   agentCRD,
	KindService: serviceCRD,
	KindFlow:    flowCRD,
}

const agentCRD = `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: agents.micro.dev
spec:
  group: micro.dev
  scope: Namespaced
  names:
    plural: agents
    singular: agent
    kind: Agent
    shortNames: [magent]
  versions:
  - name: v1alpha1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        required: [spec]
        properties:
          spec:
            type: object
            required: [image]
            properties:
              image: {type: string, minLength: 1}
              command:
                type: array
                items: {type: string}
              args:
                type: array
                items: {type: string}
              replicas: {type: integer, minimum: 0}
              registry: {type: string}
              env:
                type: object
                additionalProperties: {type: string}
`

const serviceCRD = `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: services.micro.dev
spec:
  group: micro.dev
  scope: Namespaced
  names:
    plural: services
    singular: service
    kind: Service
    shortNames: [mservice]
  versions:
  - name: v1alpha1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        required: [spec]
        properties:
          spec:
            type: object
            required: [image]
            properties:
              image: {type: string, minLength: 1}
              command:
                type: array
                items: {type: string}
              args:
                type: array
                items: {type: string}
              replicas: {type: integer, minimum: 0}
              registry: {type: string}
              env:
                type: object
                additionalProperties: {type: string}
`

const flowCRD = `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: flows.micro.dev
spec:
  group: micro.dev
  scope: Namespaced
  names:
    plural: flows
    singular: flow
    kind: Flow
    shortNames: [mflow]
  versions:
  - name: v1alpha1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        required: [spec]
        properties:
          spec:
            type: object
            required: [image]
            properties:
              image: {type: string, minLength: 1}
              command:
                type: array
                items: {type: string}
              args:
                type: array
                items: {type: string}
              replicas: {type: integer, minimum: 0}
              registry: {type: string}
              env:
                type: object
                additionalProperties: {type: string}
`
