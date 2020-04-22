package persist

import (
	"context"
	"database/sql"
	"fmt"
	"io"

	"github.com/Dynom/ERI/cmd/web/hitlist"
	"github.com/Dynom/ERI/validator"
	"github.com/Dynom/ERI/validator/validations"
	"github.com/sirupsen/logrus"
)

func New(db *sql.DB, logger logrus.FieldLogger) Persister {
	return &Postgres{
		db:     db,
		logger: logger,
	}
}

type Postgres struct {
	db     *sql.DB
	logger logrus.FieldLogger
}

func (p *Postgres) Close() error {
	return p.db.Close()
}

func (p *Postgres) Store(ctx context.Context, d hitlist.Domain, r hitlist.Recipient, vr validator.Result) error {
	stmt, err := p.db.Prepare(`
			INSERT INTO
				hitlist (domain, recipient, validations, steps)
			VALUES
				($1, $2::bytea, $3, $4)
			ON CONFLICT (domain, recipient) DO UPDATE
			SET
				validations = EXCLUDED.validations,
			  steps = EXCLUDED.steps`)

	if err != nil {
		return err
	}

	defer deferClose(stmt, p.logger)
	_, err = stmt.ExecContext(ctx, string(d), []byte(r), int64(vr.Validations), int64(vr.Steps))

	return err
}

func (p *Postgres) Range(ctx context.Context, cb PersistCallbackFn) error {

	if err := p.db.Ping(); err != nil {
		return err
	}

	stmt, err := p.db.Prepare(`
		SELECT
      domain,
      recipient::bytea,
      validations,
		  steps
		FROM
      hitlist
	`)

	if err != nil {
		return err
	}

	defer deferClose(stmt, p.logger)

	rows, err := stmt.QueryContext(ctx)
	if err != nil {
		return err
	}

	defer deferClose(rows, p.logger)

	for rows.Next() {
		var row hitListRow

		if err := rows.Scan(&row.Domain, &row.Recipient, &row.Validations, &row.Steps); err != nil {
			p.logger.WithError(err).Warn("Error scanning field")
			continue
		}

		d, r := rowToInternalParts(row)

		err := cb(d, r, validator.Result{
			Validations: validations.Validations(row.Validations),
			Steps:       validations.Steps(row.Steps),
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func rowToInternalParts(row hitListRow) (hitlist.Domain, hitlist.Recipient) {
	return hitlist.Domain(row.Domain), hitlist.Recipient(row.Recipient)
}

type hitListRow struct {
	Domain      string `sql:"domain"`
	Recipient   []byte `sql:"recipient"`
	Validations int64  `sql:"validations"`
	Steps       int64  `sql:"steps"`
}

func deferClose(toClose io.Closer, log logrus.FieldLogger) {
	if toClose == nil {
		return
	}

	err := toClose.Close()
	if err != nil {
		if log == nil {
			fmt.Printf("error failed to close handle %s", err)
			return
		}

		log.WithError(err).Error("Failed to close handle")
	}
}
