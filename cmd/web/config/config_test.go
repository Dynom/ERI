package config

import (
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

func TestDuration_AsDuration(t *testing.T) {
	type fields struct {
		duration time.Duration
	}
	tests := []struct {
		name   string
		fields fields
		want   time.Duration
	}{
		{
			name: "Testing if duration set",
			fields: fields{
				duration: time.Hour,
			},
			want: time.Hour,
		},
		{
			name: "Testing with no duration",
			fields: fields{
				duration: time.Duration(0),
			},
			want: time.Duration(0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Duration{
				duration: tt.fields.duration,
			}
			if got := d.AsDuration(); got != tt.want {
				t.Errorf("AsDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDuration_Set(t *testing.T) {
	type fields struct {
		duration time.Duration
	}
	type args struct {
		v string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Testing if durations set",
			fields: fields{
				duration: time.Hour,
			},
			args: args{
				v: "1h0m0s",
			},
			wantErr: false,
		},
		{
			name: "Testing with no duration",
			fields: fields{
				duration: time.Duration(0),
			},
			args: args{
				v: "0",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Duration{
				duration: tt.fields.duration,
			}
			if err := d.Set(tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDuration_String(t *testing.T) {
	type fields struct {
		duration time.Duration
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "Testing if duration set",
			fields: fields{
				duration: time.Hour,
			},
			want: "1h0m0s",
		},
		{
			name: "Testing with no duration",
			fields: fields{
				duration: time.Duration(0),
			},
			want: "0s",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := Duration{
				duration: tt.fields.duration,
			}
			if got := d.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDuration_UnmarshalText(t *testing.T) {
	type fields struct {
		duration time.Duration
	}
	type args struct {
		text []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Testing if duration set",
			fields: fields{
				duration: time.Hour,
			},
			args: args{
				text: []byte("60m"),
			},
			wantErr: false,
		},
		{
			name: "Testing with no duration",
			fields: fields{
				duration: time.Duration(0),
			},
			args: args{
				text: []byte("60m"),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &Duration{
				duration: tt.fields.duration,
			}
			if err := d.UnmarshalText(tt.args.text); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalText() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHeaders_Set(t *testing.T) {
	type args struct {
		v string
	}
	tests := []struct {
		name    string
		h       Headers
		want    []string
		args    args
		wantErr bool
	}{
		{
			name: "Testing if header set",
			h: map[string]string{
				"a":            "b",
				"Content-Type": "application/json",
			},
			want: []string{"a:b", "Content-Type:application/json"},
			args: args{
				v: "Content-Type:application/json",
			},
			wantErr: false,
		},
		{
			name: "Testing with empty header",
			h:    nil,
			want: []string{},
			args: args{
				v: "Content-Type:application/json",
			},
			wantErr: false,
		},
		{
			name: "Testing with invalid header",
			h:    map[string]string{},
			want: []string{},
			args: args{
				v: "test",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.h.Set(tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got := tt.h.String(); reflect.DeepEqual(got, tt.want) {
				t.Errorf("Expected Set(%q) to result in vt == %v", tt.args.v, got)
			}
		})
	}
}

func TestHeaders_String(t *testing.T) {
	tests := []struct {
		name string
		h    Headers
		want []string
	}{
		{
			name: "Testing the happy flow",
			h: map[string]string{
				"a":            "b",
				"Content-Type": "application/json",
			},
			want: []string{"a:b", "Content-Type:application/json"},
		},
		{
			name: "Testing zero value",
			h:    map[string]string{},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			// Converting to a slice and sorting, to make sure we have a consistent comparision.
			got := strings.Split(tt.h.String(), ",")
			sort.Strings(got)

			if got := tt.h.String(); reflect.DeepEqual(got, tt.want) {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLogFormat_Set(t *testing.T) {
	type args struct {
		v string
	}
	tests := []struct {
		name    string
		vt      LogFormat
		want    LogFormat
		args    args
		wantErr bool
	}{
		{
			name: "Testing if LogFormat set",
			vt:   "test",
			want: "test",
			args: args{
				v: "test",
			},
			wantErr: false,
		},
		{
			name: "Testing with empty input",
			vt:   "",
			want: "",
			args: args{
				v: "",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.vt.Set(tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.vt != tt.want {
				t.Errorf("Expected Set(%q) to result in vt == %v", tt.args.v, tt.vt)
			}
		})
	}
}

func TestLogFormat_String(t *testing.T) {
	tests := []struct {
		name string
		vt   LogFormat
		want string
	}{
		{
			name: "Testing if LogFormat set",
			vt:   "test",
			want: "test",
		},
		{
			name: "Testing with empty input",
			vt:   "",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vt.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLogFormat_UnmarshalText(t *testing.T) {
	type args struct {
		value []byte
	}
	tests := []struct {
		name    string
		vt      LogFormat
		want    LogFormat
		args    args
		wantErr bool
	}{
		{
			name: "Testing with LogFormat json",
			vt:   "json",
			want: "json",
			args: args{
				value: []byte("json"),
			},
			wantErr: false,
		},
		{
			name: "Testing with LogFormat text",
			vt:   "text",
			want: "text",
			args: args{
				value: []byte("text"),
			},
			wantErr: false,
		},
		{
			name: "Testing with unsupported value",
			vt:   "test",
			want: "test",
			args: args{
				value: []byte("test"),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.vt.UnmarshalText(tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalText() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.vt != tt.want {
				t.Errorf("Expected UnmarshalText(%q) to result in vt == %v", tt.args.value, tt.vt)
			}
		})
	}
}

func TestValidatorType_Set(t *testing.T) {
	type args struct {
		v string
	}
	tests := []struct {
		name    string
		vt      ValidatorType
		want    ValidatorType
		args    args
		wantErr bool
	}{
		{
			name: "Testing if ValidatorType set",
			vt:   "test",
			want: "test",
			args: args{
				v: "test",
			},
			wantErr: false,
		},
		{
			name: "Testing with empty input",
			vt:   "",
			want: "",
			args: args{
				v: "",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.vt.Set(tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.vt != tt.want {
				t.Errorf("Expected Set(%q) to result in vt == %v", tt.args.v, tt.vt)
			}
		})
	}
}

func TestValidatorType_String(t *testing.T) {
	tests := []struct {
		name string
		vt   ValidatorType
		want string
	}{
		{
			name: "Testing if ValidatorType set",
			vt:   "test",
			want: "test",
		},
		{
			name: "",
			vt:   "",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.vt.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidatorType_UnmarshalText(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		// The good
		{name: "Valid value", value: string(VTLookup)},

		// The bad
		{wantErr: true, name: "Invalid value", value: "Hakuna matata"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vt := ValidatorType(tt.value)

			if err := vt.UnmarshalText([]byte(tt.value)); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if _ = vt.UnmarshalText([]byte(tt.value)); string(vt) != tt.value {
				t.Errorf("UnmarshalText() value not on value receiver. Setting value %s doesn't reflect variable %v", tt.value, vt)
			}

		})
	}
}

func TestValidatorTypes_AsStringSlice(t *testing.T) {
	t.Run("alloc size test", func(t *testing.T) {
		v := ValidatorTypes{"a", "b"}
		if got := v.AsStringSlice(); cap(got) != len(got) {
			t.Errorf("Expected the capacity %d to be equal to the length %d, it wasn't.", cap(got), len(got))
		}

		if got := v.AsStringSlice(); len(got) != len(v) {
			t.Errorf("Got %d, expected a length of %d", len(got), len(v))
		}
	})
}

func TestConfig_GetSensored(t *testing.T) {

	cfg := Config{}
	cfg.Backend.URL = "test"

	const mask = "**masked**"

	exp := Config{}
	exp.Backend.URL = valueMask
	exp.Hash.Key = valueMask
	exp.Server.Profiler.Prefix = valueMask

	tests := []struct {
		name string
		c    Config
		want Config
	}{
		{
			name: "Testing with valid input",
			c:    cfg,
			want: exp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ret := tt.c.GetSensored()
			if ret.Backend != tt.want.Backend && ret.Backend.URL != valueMask {
				t.Errorf("GetSensored() got = %v, want %v", ret.Backend, tt.want.Backend)
			}
			if ret.Hash != tt.want.Hash && ret.Hash.Key != valueMask {
				t.Errorf("GetSensored() got = %v, want %v", ret.Hash, tt.want.Hash)
			}
			if ret.Server.Profiler != tt.want.Server.Profiler && ret.Server.Profiler.Prefix != valueMask {
				t.Errorf("GetSensored() got = %v, want %v", ret.Server.Profiler, tt.want.Server.Profiler)
			}
		})
	}
}

func TestString_GetSensored(t *testing.T) {

	cfg := Config{}

	tests := []struct {
		name    string
		c       Config
		want    string
		wantRet bool
	}{
		{
			name:    "Testing if string returned",
			c:       cfg,
			want:    cfg.String(),
			wantRet: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.c.String()
			if ret := strings.Contains(got, tt.want); ret != tt.wantRet {
				t.Errorf("String GetSensored() got = %v, want %v", got, tt.want)
			}
		})
	}
}
