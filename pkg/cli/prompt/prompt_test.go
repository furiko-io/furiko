/*
 * Copyright 2022 The Furiko Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package prompt_test

import (
	"context"
	"testing"
	"time"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/furiko-io/furiko/apis/execution/v1alpha1"
	"github.com/furiko-io/furiko/pkg/cli/console"
	"github.com/furiko-io/furiko/pkg/cli/prompt"
	"github.com/furiko-io/furiko/pkg/cli/streams"
	"github.com/furiko-io/furiko/pkg/utils/testutils"
)

func TestNewBoolPrompt(t *testing.T) {
	tests := []struct {
		name      string
		option    v1alpha1.Option
		procedure func(c *console.Console)
		want      interface{}
		wantErr   assert.ErrorAssertionFunc
	}{
		{
			name: "default to false",
			option: v1alpha1.Option{
				Type:  v1alpha1.OptionTypeBool,
				Name:  "example",
				Label: "Example",
			},
			procedure: func(c *console.Console) {
				c.ExpectString("Example")
				_, _ = c.Send(string(terminal.KeyEnter))
			},
			want: false,
		},
		{
			name: "default to true",
			option: v1alpha1.Option{
				Type:  v1alpha1.OptionTypeBool,
				Name:  "example",
				Label: "Example",
				Bool: &v1alpha1.BoolOptionConfig{
					Default: true,
				},
			},
			procedure: func(c *console.Console) {
				c.ExpectString("Example")
				_, _ = c.Send(string(terminal.KeyEnter))
			},
			want: true,
		},
		{
			name: "input y",
			option: v1alpha1.Option{
				Type:  v1alpha1.OptionTypeBool,
				Name:  "example",
				Label: "Example",
			},
			procedure: func(c *console.Console) {
				c.ExpectString("Example")
				c.SendLine("y")
			},
			want: true,
		},
		{
			name: "input n",
			option: v1alpha1.Option{
				Type:  v1alpha1.OptionTypeBool,
				Name:  "example",
				Label: "Example",
			},
			procedure: func(c *console.Console) {
				c.ExpectString("Example")
				c.SendLine("n")
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pt, ps, err := NewPromptTest()
			if err != nil {
				t.Fatalf("cannot initialize test: %v", err)
			}
			defer pt.Close()

			p := prompt.NewBoolPrompt(ps, tt.option)
			resp, err := pt.Run(t, p, tt.procedure)
			if testutils.WantError(t, tt.wantErr, err) {
				return
			}
			assert.Equal(t, tt.want, resp)
		})
	}
}

func TestNewStringPrompt(t *testing.T) {
	tests := []struct {
		name      string
		option    v1alpha1.Option
		procedure func(c *console.Console)
		want      interface{}
		wantErr   assert.ErrorAssertionFunc
	}{
		{
			name: "basic option",
			option: v1alpha1.Option{
				Type:  v1alpha1.OptionTypeString,
				Name:  "example",
				Label: "Example",
			},
			procedure: func(c *console.Console) {
				c.ExpectString("Example")
				c.SendLine("example_input")
			},
			want: "example_input",
		},
		{
			name: "basic option required",
			option: v1alpha1.Option{
				Type:     v1alpha1.OptionTypeString,
				Name:     "example",
				Label:    "Example",
				Required: true,
			},
			procedure: func(c *console.Console) {
				c.ExpectString("Example")
				c.SendLine("example_input")
			},
			want: "example_input",
		},
		{
			name: "accept empty input",
			option: v1alpha1.Option{
				Type:  v1alpha1.OptionTypeString,
				Name:  "example",
				Label: "Example",
			},
			procedure: func(c *console.Console) {
				c.ExpectString("Example")
				c.SendLine("")
			},
			want: "",
		},
		{
			name: "fail on empty input if required",
			option: v1alpha1.Option{
				Type:     v1alpha1.OptionTypeString,
				Name:     "example",
				Label:    "Example",
				Required: true,
			},
			procedure: func(c *console.Console) {
				c.ExpectString("Example")
				c.SendLine("")
			},
			wantErr: testutils.AssertErrorIs(context.DeadlineExceeded),
		},
		{
			name: "use default value",
			option: v1alpha1.Option{
				Type:     v1alpha1.OptionTypeString,
				Name:     "example",
				Label:    "Example",
				Required: true,
				String: &v1alpha1.StringOptionConfig{
					Default: "default_value",
				},
			},
			procedure: func(c *console.Console) {
				c.ExpectString("Example")
				c.SendLine("")
			},
			want: "default_value",
		},
		{
			name: "use input with default value",
			option: v1alpha1.Option{
				Type:     v1alpha1.OptionTypeString,
				Name:     "example",
				Label:    "Example",
				Required: true,
				String: &v1alpha1.StringOptionConfig{
					Default: "default_value",
				},
			},
			procedure: func(c *console.Console) {
				c.ExpectString("Example")
				c.SendLine("example_input")
			},
			want: "example_input",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pt, ps, err := NewPromptTest()
			if err != nil {
				t.Fatalf("cannot initialize test: %v", err)
			}
			defer pt.Close()

			p := prompt.NewStringPrompt(ps, tt.option)
			resp, err := pt.Run(t, p, tt.procedure)
			if testutils.WantError(t, tt.wantErr, err) {
				return
			}
			assert.Equal(t, tt.want, resp)
		})
	}
}

func TestNewSelectPrompt(t *testing.T) {
	tests := []struct {
		name      string
		option    v1alpha1.Option
		procedure func(c *console.Console)
		want      interface{}
		wantErr   assert.ErrorAssertionFunc
	}{
		{
			name: "basic option",
			option: v1alpha1.Option{
				Type:  v1alpha1.OptionTypeSelect,
				Name:  "example",
				Label: "Example",
				Select: &v1alpha1.SelectOptionConfig{
					Values: []string{"apple", "banana", "orange"},
				},
			},
			procedure: func(c *console.Console) {
				c.ExpectString("Example")
				c.SendLine("orange")
			},
			want: "orange",
		},
		{
			name: "custom option without AllowCustom",
			option: v1alpha1.Option{
				Type:  v1alpha1.OptionTypeSelect,
				Name:  "example",
				Label: "Example",
				Select: &v1alpha1.SelectOptionConfig{
					Values: []string{"apple", "banana", "orange"},
				},
			},
			procedure: func(c *console.Console) {
				c.ExpectString("Example")
				c.SendLine("invalid_option")
			},
			wantErr: testutils.AssertErrorIs(context.DeadlineExceeded),
		},
		{
			name: "custom option with AllowCustom",
			option: v1alpha1.Option{
				Type:  v1alpha1.OptionTypeSelect,
				Name:  "example",
				Label: "Example",
				Select: &v1alpha1.SelectOptionConfig{
					Values:      []string{"apple", "banana", "orange"},
					AllowCustom: true,
				},
			},
			procedure: func(c *console.Console) {
				c.ExpectString("Example")
				c.SendLine("custom_option")
			},
			want: "custom_option",
		},
		{
			name: "enter with default value",
			option: v1alpha1.Option{
				Type:  v1alpha1.OptionTypeSelect,
				Name:  "example",
				Label: "Example",
				Select: &v1alpha1.SelectOptionConfig{
					Values:  []string{"apple", "banana", "orange"},
					Default: "orange",
				},
			},
			procedure: func(c *console.Console) {
				c.ExpectString("Example")
				_, _ = c.Send(string(terminal.KeyEnter))
			},
			want: "orange",
		},
		{
			name: "using arrow keys",
			option: v1alpha1.Option{
				Type:  v1alpha1.OptionTypeSelect,
				Name:  "example",
				Label: "Example",
				Select: &v1alpha1.SelectOptionConfig{
					Values: []string{"apple", "banana", "orange"},
				},
			},
			procedure: func(c *console.Console) {
				c.ExpectString("Example")
				_, _ = c.Send(string(terminal.KeyArrowDown))
				_, _ = c.Send(string(terminal.KeyEnter))
			},
			want: "banana",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pt, ps, err := NewPromptTest()
			if err != nil {
				t.Fatalf("cannot initialize test: %v", err)
			}
			defer pt.Close()

			p := prompt.NewSelectPrompt(ps, tt.option)
			resp, err := pt.Run(t, p, tt.procedure)
			if testutils.WantError(t, tt.wantErr, err) {
				return
			}
			assert.Equal(t, tt.want, resp)
		})
	}
}

func TestNewMultiPrompt(t *testing.T) {
	tests := []struct {
		name      string
		option    v1alpha1.Option
		procedure func(c *console.Console)
		want      interface{}
		wantErr   assert.ErrorAssertionFunc
	}{
		{
			name: "select no options",
			option: v1alpha1.Option{
				Type:  v1alpha1.OptionTypeMulti,
				Name:  "example",
				Label: "Example",
				Multi: &v1alpha1.MultiOptionConfig{
					Values: []string{"apple", "banana", "orange"},
				},
			},
			procedure: func(c *console.Console) {
				c.ExpectString("Example")
				_, _ = c.Send(string(terminal.KeyEnter))
			},
			want: []string(nil),
		},
		{
			name: "select no options but required",
			option: v1alpha1.Option{
				Type:     v1alpha1.OptionTypeMulti,
				Name:     "example",
				Label:    "Example",
				Required: true,
				Multi: &v1alpha1.MultiOptionConfig{
					Values: []string{"apple", "banana", "orange"},
				},
			},
			procedure: func(c *console.Console) {
				c.ExpectString("Example")
				_, _ = c.Send(string(terminal.KeyEnter))
			},
			wantErr: testutils.AssertErrorIs(context.DeadlineExceeded),
		},
		{
			name: "select single option",
			option: v1alpha1.Option{
				Type:  v1alpha1.OptionTypeMulti,
				Name:  "example",
				Label: "Example",
				Multi: &v1alpha1.MultiOptionConfig{
					Values: []string{"apple", "banana", "orange"},
				},
			},
			procedure: func(c *console.Console) {
				c.ExpectString("Example")
				_, _ = c.Send(string(terminal.KeySpace))
				_, _ = c.Send(string(terminal.KeyEnter))
			},
			want: []string{"apple"},
		},
		{
			name: "use default values",
			option: v1alpha1.Option{
				Type:  v1alpha1.OptionTypeMulti,
				Name:  "example",
				Label: "Example",
				Multi: &v1alpha1.MultiOptionConfig{
					Values:  []string{"apple", "banana", "orange"},
					Default: []string{"banana", "orange"},
				},
			},
			procedure: func(c *console.Console) {
				c.ExpectString("Example")
				_, _ = c.Send(string(terminal.KeyEnter))
			},
			want: []string{"banana", "orange"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pt, ps, err := NewPromptTest()
			if err != nil {
				t.Fatalf("cannot initialize test: %v", err)
			}
			defer pt.Close()

			p := prompt.NewMultiPrompt(ps, tt.option)
			resp, err := pt.Run(t, p, tt.procedure)
			if testutils.WantError(t, tt.wantErr, err) {
				return
			}
			assert.Equal(t, tt.want, resp)
		})
	}
}

func TestNewDatePrompt(t *testing.T) {
	tests := []struct {
		name      string
		option    v1alpha1.Option
		procedure func(c *console.Console)
		want      interface{}
		wantErr   assert.ErrorAssertionFunc
	}{
		{
			name: "empty value",
			option: v1alpha1.Option{
				Type:  v1alpha1.OptionTypeDate,
				Name:  "example",
				Label: "Example",
			},
			procedure: func(c *console.Console) {
				c.ExpectString("Example")
				_, _ = c.Send(string(terminal.KeyEnter))
			},
			want: "",
		},
		{
			name: "empty value but required",
			option: v1alpha1.Option{
				Type:     v1alpha1.OptionTypeDate,
				Name:     "example",
				Label:    "Example",
				Required: true,
			},
			procedure: func(c *console.Console) {
				c.ExpectString("Example")
				_, _ = c.Send(string(terminal.KeyEnter))
			},
			wantErr: testutils.AssertErrorIs(context.DeadlineExceeded),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pt, ps, err := NewPromptTest()
			if err != nil {
				t.Fatalf("cannot initialize test: %v", err)
			}
			defer pt.Close()

			p := prompt.NewDatePrompt(ps, tt.option)
			resp, err := pt.Run(t, p, tt.procedure)
			if testutils.WantError(t, tt.wantErr, err) {
				return
			}
			assert.Equal(t, tt.want, resp)
		})
	}
}

type PromptTest struct {
	console *console.Console
}

func NewPromptTest() (*PromptTest, *streams.Streams, error) {
	iostreams, _, _, _ := genericclioptions.NewTestIOStreams()
	c, err := console.NewConsole(iostreams.Out)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to init pseudo TTY")
	}

	ps := streams.NewTTYStreams(c.Tty())
	pt := &PromptTest{
		console: c,
	}

	return pt, ps, nil
}

func (pt *PromptTest) Run(t *testing.T, p prompt.Prompt, procedure func(c *console.Console)) (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
	defer cancel()

	done := pt.console.Run(procedure)

	type response struct {
		retval interface{}
		err    error
	}
	respChan := make(chan response)

	go func() {
		// Run the prompt.
		// Note that this will block if the validation fails.
		output, err := p.Run()
		respChan <- response{retval: output, err: err}
	}()

	var resp response
	select {
	case <-ctx.Done():
		resp.err = ctx.Err()
		if _, err := pt.console.Send(string(terminal.KeyInterrupt)); err != nil {
			t.Errorf("error sending interrupt: %v", err)
		}
	case ret := <-respChan:
		resp = ret
	}

	// Wait for tty to be closed.
	if err := pt.console.Tty().Close(); err != nil {
		t.Errorf("error closing tty: %v", err)
	}
	<-done

	return resp.retval, resp.err
}

func (pt *PromptTest) Close() {
	_ = pt.console.Close()
}
