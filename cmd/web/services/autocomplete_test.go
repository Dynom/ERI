package services

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/Dynom/ERI/cmd/web/hitlist"
	hlTest "github.com/Dynom/ERI/cmd/web/hitlist/test"
	"github.com/Dynom/ERI/validator"
	"github.com/Dynom/ERI/validator/validations"
	"github.com/Dynom/TySug/finder"
	"github.com/sirupsen/logrus"
	lrTest "github.com/sirupsen/logrus/hooks/test"
)

func TestAutocompleteSvc_Autocomplete(t *testing.T) {

	ctxExpired, cancel := context.WithTimeout(context.Background(), -1*time.Hour)
	cancel()

	type fields struct {
		recipientThreshold uint64
	}

	type args struct {
		ctx    context.Context
		domain string
		limit  uint64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    AutocompleteResult
		wantErr bool
	}{
		{
			name:   "Happy flow",
			fields: fields{recipientThreshold: 1},
			args: args{
				ctx:    context.Background(),
				domain: "gm",
				limit:  2,
			},
			want: AutocompleteResult{
				Suggestions: []string{"gmail.2"},
			},
			wantErr: false,
		},
		{
			name:   "Hitting list limit",
			fields: fields{recipientThreshold: 0},
			args: args{
				ctx:    context.Background(),
				domain: "gm",
				limit:  1,
			},
			want: AutocompleteResult{
				Suggestions: []string{"gmail.2"},
			},
			wantErr: false,
		},
		{
			name:   "Expired context",
			fields: fields{recipientThreshold: 1},
			args: args{
				ctx:    ctxExpired,
				domain: "gm",
				limit:  2,
			},
			want:    AutocompleteResult{},
			wantErr: true,
		},
	}

	hl := hitlist.New(hlTest.MockHasher{}, 1*time.Hour)
	_ = hl.AddDomain("example", validator.Result{
		Validations: validations.Validations(validations.FSyntax | validations.FMXLookup | validations.FValid),
		Steps:       validations.Steps(validations.FSyntax),
	})

	_ = hl.AddDomain("gmail.0", validator.Result{
		Validations: validations.Validations(validations.FSyntax | validations.FMXLookup | validations.FValid),
		Steps:       validations.Steps(validations.FSyntax),
	})

	_ = hl.AddEmailAddress("john.doe@gmail.2", validator.Result{
		Validations: validations.Validations(validations.FSyntax | validations.FMXLookup | validations.FValid),
		Steps:       validations.Steps(validations.FSyntax),
	})

	_ = hl.AddEmailAddress("jane.doe@gmail.2", validator.Result{
		Validations: validations.Validations(validations.FSyntax | validations.FMXLookup | validations.FValid),
		Steps:       validations.Steps(validations.FSyntax),
	})

	f, err := finder.New(hl.GetValidAndUsageSortedDomains(), finder.WithAlgorithm(finder.NewJaroWinklerDefaults()))
	if err != nil {
		t.Errorf("Setting up test failed.")
		t.FailNow()
	}

	logger, hook := lrTest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook.Reset()
			a := NewAutocompleteService(f, hl, tt.fields.recipientThreshold, logger)

			got, err := a.Autocomplete(tt.args.ctx, tt.args.domain, tt.args.limit)
			if (err != nil) != tt.wantErr {
				t.Errorf("Autocomplete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Autocomplete() got = %v, want %v", got, tt.want)
				t.Logf("logs %+v", hook.AllEntries())
			}
		})
	}
}

func TestAutocompleteSvc_filter(t *testing.T) {

	ctxExpired, cancel := context.WithTimeout(context.Background(), -1*time.Hour)
	cancel()

	hl := hitlist.New(hlTest.MockHasher{}, 1*time.Hour)
	_ = hl.AddDomain("example", validator.Result{
		Validations: validations.Validations(validations.FSyntax | validations.FMXLookup | validations.FValid),
		Steps:       validations.Steps(validations.FSyntax),
	})

	_ = hl.AddDomain("gmail.0", validator.Result{
		Validations: validations.Validations(validations.FSyntax | validations.FMXLookup | validations.FValid),
		Steps:       validations.Steps(validations.FSyntax),
	})

	_ = hl.AddEmailAddress("john.doe@gmail.2", validator.Result{
		Validations: validations.Validations(validations.FSyntax | validations.FMXLookup | validations.FValid),
		Steps:       validations.Steps(validations.FSyntax),
	})

	_ = hl.AddEmailAddress("jane.doe@gmail.2", validator.Result{
		Validations: validations.Validations(validations.FSyntax | validations.FMXLookup | validations.FValid),
		Steps:       validations.Steps(validations.FSyntax),
	})

	type fields struct {
		finder             *finder.Finder
		logger             logrus.FieldLogger
		hitList            *hitlist.HitList
		recipientThreshold uint64
	}
	type args struct {
		ctx   context.Context
		list  []string
		limit uint64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "Happy flow",
			fields: fields{
				recipientThreshold: 0,
			},
			args: args{
				ctx:   context.Background(),
				list:  hl.GetValidAndUsageSortedDomains(),
				limit: 2,
			},
			want:    hl.GetValidAndUsageSortedDomains()[0:2],
			wantErr: false,
		},
		{
			name: "Expired context",
			fields: fields{
				recipientThreshold: 0,
			},
			args: args{
				ctx:   ctxExpired,
				list:  hl.GetValidAndUsageSortedDomains(),
				limit: 2,
			},
			want:    []string{},
			wantErr: true,
		},
	}

	f, err := finder.New(hl.GetValidAndUsageSortedDomains(), finder.WithAlgorithm(finder.NewJaroWinklerDefaults()))
	if err != nil {
		t.Errorf("Setting up test failed.")
		t.FailNow()
	}

	logger, hook := lrTest.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook.Reset()

			a := NewAutocompleteService(f, hl, tt.fields.recipientThreshold, logger)

			got, err := a.filter(tt.args.ctx, tt.args.list, tt.args.limit)
			if (err != nil) != tt.wantErr {
				t.Errorf("filter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filter() got = %v, want %v", got, tt.want)
			}
		})
	}
}
