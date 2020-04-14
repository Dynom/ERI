package config

import (
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
		// TODO: Add test cases.
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
		// TODO: Add test cases.
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
		// TODO: Add test cases.
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
		// TODO: Add test cases.
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
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.h.Set(tt.args.v); (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHeaders_String(t *testing.T) {
	tests := []struct {
		name string
		h    Headers
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.h.String(); got != tt.want {
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
			name: "test",
			vt:   "test",
			want: "test",
			args: args{
				v: "test",
			},
			wantErr: false,
		},
		{
			name: "",
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
			name: "test",
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
			name: "test",
			vt:   "test",
			want: "json",
			args: args{
				value: []byte("json"),
			},
			wantErr: false,
		},
		{
			name: "",
			vt:   "",
			want: "json",
			args: args{
				value: []byte("json"),
			},
			wantErr: false,
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
			name: "test",
			vt:   "test",
			want: "test",
			args: args{
				v: "test",
			},
			wantErr: false,
		},
		{
			name: "",
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
			name: "test",
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
