// Package flow is maintained for backward compatibility.
// The canonical import is go-micro.dev/v6/flow.
package flow

import "go-micro.dev/v6/flow"

// Re-export types for backward compatibility.
type Flow = flow.Flow
type Options = flow.Options
type Option = flow.Option
type Result = flow.Result

var New = flow.New
var Trigger = flow.Trigger
var Prompt = flow.Prompt
var SystemPrompt = flow.SystemPrompt
var Provider = flow.Provider
var APIKey = flow.APIKey
var Model = flow.Model
var BaseURL = flow.BaseURL
var HistoryLimit = flow.HistoryLimit
var OnResult = flow.OnResult
