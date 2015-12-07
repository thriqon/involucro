package file

import (
	"github.com/Shopify/go-lua"
	log "github.com/Sirupsen/logrus"
	"github.com/fsouza/go-dockerclient"
	runS "github.com/thriqon/involucro/steps/run"
	"path"
	"regexp"
	"strings"
)

type usingBuilderState struct {
	builderState
	runS.ExecuteImage
}

func (bs builderState) using(l *lua.State) int {
	nbs := usingBuilderState{
		builderState: bs,
		ExecuteImage: runS.ExecuteImage{
			Config: docker.Config{
				Image: requireStringOrFailGracefully(l, -1, "using"),
			},
			HostConfig: docker.HostConfig{
				Binds: []string{
					"./:/source",
				},
			},
		},
	}
	return usingTable(l, &nbs)
}

func (ubs usingBuilderState) usingRun(l *lua.State) int {
	ubs.Config.Cmd = argumentsToStringArray(l)
	if ubs.Config.WorkingDir == "" {
		ubs.Config.WorkingDir = "/source"
	}

	ubs.HostConfig = absolutizeBinds(ubs.HostConfig, ubs.inv.WorkingDir)

	tasks := ubs.inv.Tasks
	tasks[ubs.taskID] = append(tasks[ubs.taskID], ubs)

	return usingTable(l, &ubs)
}

func usingTable(l *lua.State, ubs *usingBuilderState) int {
	return tableWith(l, fm{
		"using":           ubs.using,
		"run":             ubs.usingRun,
		"task":            ubs.task,
		"wrap":            ubs.wrap,
		"withExpectation": ubs.usingWithExpectation,
		"withConfig":      ubs.withConfig,
		"withHostConfig":  ubs.withHostConfig,
	})
}

func (nubs usingBuilderState) usingWithExpectation(l *lua.State) int {
	if l.Top() != 1 {
		lua.Errorf(l, "expected exactly one argument to 'withExpectation'")
		panic("unreachable")
	}
	lua.ArgumentCheck(l, l.IsTable(-1), 1, "Expected table as argument")

	l.Field(-1, "code")
	if !l.IsNil(-1) {
		nubs.ExpectedCode = lua.CheckInteger(l, -1)
		log.WithFields(log.Fields{"code": nubs.ExpectedCode}).Info("Expecting code")
	}
	l.Pop(1)

	l.Field(-1, "stdout")
	if !l.IsNil(-1) {
		str := lua.CheckString(l, -1)
		if regex, err := regexp.Compile(str); err != nil {
			lua.ArgumentError(l, 1, "invalid regular expression in stdout: "+err.Error())
			panic("unreachable")
		} else {
			nubs.ExpectedStdoutMatcher = regex
		}
	}
	l.Pop(1)

	l.Field(-1, "stderr")
	if !l.IsNil(-1) {
		str := lua.CheckString(l, -1)
		if regex, err := regexp.Compile(str); err != nil {
			lua.ArgumentError(l, 1, "invalid regular expression in stderr: "+err.Error())
			panic("unreachable")
		} else {
			nubs.ExpectedStderrMatcher = regex
		}
	}
	l.Pop(1)

	return usingTable(l, &nubs)
}

func absolutizeBinds(h docker.HostConfig, workDir string) docker.HostConfig {
	for ind, el := range h.Binds {
		parts := strings.Split(el, ":")
		if len(parts) != 2 {
			log.WithFields(log.Fields{"bind": el}).Panic("Invalid bind, has to be of the form: source:dest")
		}

		if !path.IsAbs(parts[0]) {
			h.Binds[ind] = path.Join(workDir, parts[0]) + ":" + parts[1]
		}
	}
	return h
}
