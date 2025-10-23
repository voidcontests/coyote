package submission

import (
	"context"
	"fmt"
	"time"

	"github.com/voidcontests/coyote/internal/domain"
	"github.com/voidcontests/coyote/internal/domain/verdict"
	"github.com/voidcontests/coyote/pkg/container"
	"github.com/voidcontests/coyote/pkg/judge"
	"github.com/voidcontests/coyote/pkg/language"
)

type report struct {
	verdict          string
	passed           int
	total            int
	stderr           string
	failedTestCase   *domain.TestCase
	failedTestOutput string
}

func (s *Service) runTests(ctx context.Context, code string, l language.Language, tcs []domain.TestCase) (report, error) {
	cc, err := container.New(ctx, s.client)
	if err != nil {
		return report{}, err
	}
	defer cc.Flush(ctx)

	path := struct {
		source string
		build  string
		input  string
	}{
		source: fmt.Sprintf("/sandbox/solution.%s", l.Extension),
		build:  "/sandbox/solution",
		input:  "/sandbox/input.txt",
	}

	err = cc.WriteFile(ctx, path.source, code)
	if err != nil {
		return report{}, err
	}

	tt := len(tcs)
	if l.IsCompiled {
		cmd, ok := language.GetCompilationCommand(l, path.source, path.build)
		if !ok {
			return report{}, fmt.Errorf("no compilation command for language: %s", l.Name)
		}

		pr, err := cc.Execute(ctx, cmd)
		if err != nil {
			return report{}, err
		}

		if !pr.Ok {
			return report{
				verdict:          verdict.CE,
				passed:           0,
				total:            tt,
				stderr:           pr.Stderr,
				failedTestCase:   nil,
				failedTestOutput: "",
			}, nil
		}
	}

	cmd, ok := language.GetExecutionCommand(l, path.source, path.input, path.build)
	if !ok {
		return report{}, fmt.Errorf("no execution command for language: %s", l.Name)
	}

	for i, tc := range tcs {
		err = cc.WriteFile(ctx, path.input, tc.Input)
		if err != nil {
			return report{}, err
		}

		pr, err := s.executeWithTimeout(ctx, cc, cmd, 2*time.Second)
		if err == context.DeadlineExceeded {
			return report{
				verdict:        verdict.TLE,
				passed:         i,
				total:          tt,
				failedTestCase: &tc,
			}, nil
		}
		if err != nil {
			return report{}, err
		}

		if !pr.Ok {
			return report{
				verdict:          verdict.RE,
				passed:           i,
				total:            tt,
				stderr:           pr.Stderr,
				failedTestCase:   &tc,
				failedTestOutput: pr.Stdout,
			}, nil
		}

		// TODO: Introduce judge message for testing report
		jr := judge.Tokens(pr.Stdout, tc.Output)
		if jr.Verdict != verdict.OK {
			return report{
				verdict:          jr.Verdict,
				passed:           i,
				total:            tt,
				stderr:           pr.Stderr,
				failedTestCase:   &tc,
				failedTestOutput: pr.Stdout,
			}, nil
		}
	}

	return report{
		verdict:          verdict.OK,
		passed:           tt,
		total:            tt,
		stderr:           "",
		failedTestCase:   nil,
		failedTestOutput: "",
	}, nil
}

func (s *Service) executeWithTimeout(ctx context.Context, cc *container.Context, cmd string, timeout time.Duration) (container.ProcessResult, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	prch := make(chan container.ProcessResult, 1)
	errch := make(chan error, 1)

	go func() {
		pr, err := cc.Execute(ctx, cmd)
		if err != nil {
			errch <- err
			return
		}
		prch <- pr
	}()

	select {
	case pr := <-prch:
		return pr, nil
	case err := <-errch:
		return container.ProcessResult{}, err
	case <-ctx.Done():
		return container.ProcessResult{}, context.DeadlineExceeded
	}
}
