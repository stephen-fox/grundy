package results

import (
	"bytes"
)

const (
	DeleteShortcut Operation = "delete shortcut"
	UpdateShortcut Operation = "update_shortcut"
	CreateShortcut Operation = "create_shortcut"
)

type Operation string

func (o Operation) String() string {
	return string(o)
}

const (
	Succeeded            Outcome = "succeeded"
	SucceededWithWarning Outcome = "succeeded with warning(s)"
	Failed               Outcome = "failed"
	Skipped              Outcome = "skipped"
)

type Outcome string

func (o Outcome) String() string {
	return string(o)
}

type Result interface {
	PrintableResult() string
	Operation() Operation
	Outcome() Outcome
	GameName() string
	SteamUserId() string
	Reason() string
}

type defaultResult struct {
	operation Operation
	result    Outcome
	gameName  string
	userId    string
	reason    string
}

func (o *defaultResult) PrintableResult() string {
	buffer := bytes.NewBuffer(nil)
	buffer.WriteString("Operation ")
	buffer.WriteString(o.operation.String())
	buffer.WriteString(" has ")
	buffer.WriteString(o.result.String())
	if len(o.gameName) > 0 {
		buffer.WriteString(" for game '")
		buffer.WriteString(o.gameName)
		buffer.WriteString("'")
	}
	if len(o.userId) > 0 {
		buffer.WriteString(" for Steam user ID '")
		buffer.WriteString(o.userId)
		buffer.WriteString("'")
	}
	if len(o.reason) > 0 {
		buffer.WriteString(" - ")
		buffer.WriteString(o.reason)
	}

	return buffer.String()
}

func (o *defaultResult) Operation() Operation {
	return o.operation
}

func (o *defaultResult) Outcome() Outcome {
	return o.result
}

func (o *defaultResult) GameName() string {
	return o.gameName
}

func (o *defaultResult) SteamUserId() string {
	return o.userId
}

func (o *defaultResult) Reason() string {
	return o.reason
}

func NewDeleteSteamUserShortcutSuccess(gameName string, userId string, reason string) Result {
	return &defaultResult{
		operation: DeleteShortcut,
		result:    Succeeded,
		gameName:  gameName,
		userId:    userId,
		reason:    reason,
	}
}

func NewDeleteSteamUserShortcutSuccessWarning(gameName string, userId string, reason string) Result {
	return &defaultResult{
		operation: DeleteShortcut,
		result:    SucceededWithWarning,
		gameName:  gameName,
		userId:    userId,
		reason:    reason,
	}
}

func NewDeleteSteamUserShortcutFailure(gameName string, userId string, reason string) Result {
	return &defaultResult{
		operation: DeleteShortcut,
		result:    Failed,
		gameName:  gameName,
		userId:    userId,
		reason:    reason,
	}
}

func NewDeleteSteamUserShortcutSkipped(gameName string, userId string, reason string) Result {
	return &defaultResult{
		operation: DeleteShortcut,
		result:    Skipped,
		gameName:  gameName,
		userId:    userId,
		reason:    reason,
	}
}

func NewDeleteShortcutSkipped(gameName string, reason string) Result {
	return &defaultResult{
		operation: DeleteShortcut,
		result:    Skipped,
		gameName:  gameName,
		reason:    reason,
	}
}

func NewUpdateShortcutSkipped(gameName string, reason string) Result {
	return &defaultResult{
		operation: UpdateShortcut,
		result:    Skipped,
		gameName:  gameName,
		reason:    reason,
	}
}

func NewUpdateShortcutFailed(gameName string, reason string) Result {
	return &defaultResult{
		operation: UpdateShortcut,
		result:    Failed,
		gameName:  gameName,
		reason:    reason,
	}
}

func NewUpdateSteamUserShortcutFailed(gameName string, userId string, reason string) Result {
	return &defaultResult{
		operation: UpdateShortcut,
		result:    Failed,
		gameName:  gameName,
		userId:    userId,
		reason:    reason,
	}
}

func NewUpdateShortcutSuccess(gameName string) Result {
	return &defaultResult{
		operation: UpdateShortcut,
		result:    Succeeded,
		gameName:  gameName,
	}
}

func NewCreateShortcutSuccess(gameName string) Result {
	return &defaultResult{
		operation: CreateShortcut,
		result:    Succeeded,
		gameName:  gameName,
	}
}

func NewCreateShortcutSuccessWithWarnings(gameName string, reason string) Result {
	return &defaultResult{
		operation: CreateShortcut,
		result:    SucceededWithWarning,
		gameName:  gameName,
		reason:    reason,
	}
}
