package cfg

import (
	"github.com/simplesurance/baur/v1/cfg/resolver"
)

// Task is a task section
type Task struct {
	Name     string   `toml:"name" comment:"Identifies the task, currently the name must be 'build'."`
	Command  []string `toml:"command" comment:"Command to execute.\n The first element is the command, the following it's arguments.\n If the command element contains no path seperators,\n the path is looked up via the $PATH environment variable."`
	Includes []string `toml:"includes" comment:"Input or Output includes that the task inherits.\n Includes are specified in the format <filepath>#<ID>.\n Paths are relative to the application directory.\n Valid variables: $ROOT."`
	Input    Input    `toml:"Input" comment:"Specification of task inputs like source files, Makefiles, etc"`
	Output   Output   `toml:"Output" comment:"Specification of task outputs produced by the Task.command"`
}

func (t *Task) GetCommand() []string {
	return t.Command
}
func (t *Task) GetName() string {
	return t.Name
}

func (t *Task) GetIncludes() *[]string {
	return &t.Includes
}

func (t *Task) GetInput() *Input {
	return &t.Input
}

func (t *Task) GetOutput() *Output {
	return &t.Output
}

func (t *Task) Resolve(resolvers resolver.Resolver) error {
	var err error

	for i, elem := range t.Command {
		if t.Command[i], err = resolvers.Resolve(elem); err != nil {
			return FieldErrorWrap(err, "Command")
		}
	}

	if err := t.Input.Resolve(resolvers); err != nil {
		return FieldErrorWrap(err, "Input")
	}

	if err := t.Output.Resolve(resolvers); err != nil {
		return FieldErrorWrap(err, "Output")
	}

	return nil
}
