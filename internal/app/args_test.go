package app

import "testing"

const testCommand = "test"

func TestParseCommandArgs_EmptyInput(t *testing.T) {
	t.Parallel()
	gotCommand, gotArgs, gotArgv := ParseCommandArgs("")

	if gotCommand != "" {
		t.Errorf("ParseCommandArgs(\"\") command = %q, want empty string", gotCommand)
	}
	if gotArgs != "" {
		t.Errorf("ParseCommandArgs(\"\") args = %q, want empty string", gotArgs)
	}
	if len(gotArgv) != 0 {
		t.Errorf("ParseCommandArgs(\"\") argv = %v, want empty slice", gotArgv)
	}
}

func TestParseCommandArgs_CommandOnly(t *testing.T) {
	t.Parallel()
	gotCommand, gotArgs, gotArgv := ParseCommandArgs("test")

	if gotCommand != testCommand {
		t.Errorf("ParseCommandArgs(\"test\") command = %q, want \"test\"", gotCommand)
	}
	if gotArgs != "" {
		t.Errorf("ParseCommandArgs(\"test\") args = %q, want empty string", gotArgs)
	}
	wantArgv := []string{"test"}
	if len(gotArgv) != len(wantArgv) || gotArgv[0] != wantArgv[0] {
		t.Errorf("ParseCommandArgs(\"test\") argv = %v, want %v", gotArgv, wantArgv)
	}
}

func TestParseCommandArgs_SimpleArguments(t *testing.T) {
	t.Parallel()
	gotCommand, gotArgs, gotArgv := ParseCommandArgs("test foo bar")

	if gotCommand != testCommand {
		t.Errorf("ParseCommandArgs(\"test foo bar\") command = %q, want \"test\"", gotCommand)
	}
	if gotArgs != "foo bar" {
		t.Errorf("ParseCommandArgs(\"test foo bar\") args = %q, want \"foo bar\"", gotArgs)
	}
	wantArgv := []string{"test", "foo", "bar"}
	if len(gotArgv) != len(wantArgv) {
		t.Errorf("ParseCommandArgs(\"test foo bar\") argv length = %d, want %d", len(gotArgv), len(wantArgv))
	} else {
		for i, want := range wantArgv {
			if gotArgv[i] != want {
				t.Errorf("ParseCommandArgs(\"test foo bar\") argv[%d] = %q, want %q", i, gotArgv[i], want)
			}
		}
	}
}

func TestParseCommandArgs_QuotedArguments(t *testing.T) {
	t.Parallel()
	gotCommand, gotArgs, gotArgv := ParseCommandArgs("test foo \"bar baz\" qux")

	if gotCommand != testCommand {
		t.Errorf("ParseCommandArgs() command = %q, want \"test\"", gotCommand)
	}
	if gotArgs != "foo \"bar baz\" qux" {
		t.Errorf("ParseCommandArgs() args = %q, want \"foo \\\"bar baz\\\" qux\"", gotArgs)
	}
	wantArgv := []string{"test", "foo", "bar baz", "qux"}
	if len(gotArgv) != len(wantArgv) {
		t.Errorf("ParseCommandArgs() argv length = %d, want %d", len(gotArgv), len(wantArgv))
	} else {
		for i, want := range wantArgv {
			if gotArgv[i] != want {
				t.Errorf("ParseCommandArgs() argv[%d] = %q, want %q", i, gotArgv[i], want)
			}
		}
	}
}
