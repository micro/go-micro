package template

var (
	Readme = `# {{title .Alias}} Service

This is the {{title .Alias}} service

Generated with

` + "```" +
		`
micro new {{.Alias}}
` + "```" + `

## Usage

Generate the proto code

` + "```" +
		`
make proto
` + "```" + `

Run the service

` + "```" +
		`
micro run .
` + "```"
)
