package main

import (
	"database/sql"
	"time"

	"github.com/Dynom/ERI/validator"

	"github.com/sirupsen/logrus"

	"github.com/Dynom/ERI/validator/validations"

	"github.com/Dynom/ERI/cmd/web/hitlist"
)

type hitListRow struct {
	Domain      string                  `sql:"domain"`
	Recipient   string                  `sql:"recipient"`
	Validations validations.Validations `sql:"validations"`
	Steps       validations.Steps       `sql:"steps"`
}

func preloadValues(conn *sql.DB, list *hitlist.HitList, logger logrus.FieldLogger) (uint, error) {

	if err := conn.Ping(); err != nil {
		return 0, err
	}

	var now = time.Now()
	stmt, err := conn.Prepare("SELECT domain, recipient, validations FROM hitlist")
	if err != nil {
		return 0, err
	}

	rows, err := stmt.Query()
	if err != nil {
		return 0, err
	}

	var dbCollected uint
	for rows.Next() {
		var row hitListRow

		if err := rows.Scan(&row.Domain, &row.Recipient, &row.Validations); err != nil {
			logger.WithError(err).Error("Error scanning field")
			continue
		}

		logger := logger.WithField("row", row)
		vr := validator.Result{
			Validations: row.Validations,
			Steps:       row.Steps,
		}

		var err error
		if row.Recipient == "" {
			err = list.AddDomain(row.Domain, vr)
		} else {
			err = list.AddEmailAddress(row.Recipient+`@`+row.Domain, vr)
		}

		if err != nil {
			logger.WithError(err).Error("Error Adding e-mail address / domain to hit list")
			continue
		}

		dbCollected++
	}

	logger.WithField("time_µs", time.Since(now)).Debug("Done loading from backend")

	return dbCollected, nil
}

func registerPersistCallback(conn *sql.DB, list *hitlist.HitList, logger logrus.FieldLogger) {

	stmt, err := conn.Prepare(`
			INSERT INTO hitlist (domain, recipient, validations, steps)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (domain, recipient) DO UPDATE
			SET validations = EXCLUDED.validations,
			    steps = EXCLUDED.steps`)

	if err != nil {
		panic(err)
	}

	list.RegisterOnChange(func(r hitlist.Recipient, d string, vr validator.Result, c hitlist.ChangeType) {
		if c != hitlist.ChangeAdd {
			return
		}

		_, err := stmt.Exec(d, r.String(), vr.Validations, vr.Steps)
		if err != nil {
			logger.WithError(err).Error("Couldn't persist change")
		} else {
			logger.WithFields(logrus.Fields{
				"domain":      d,
				"recipient":   r.String(),
				"validations": vr.Validations.String(),
				"steps":       vr.Steps.String(),
			}).Debug("Persisted (new) recipient")
		}
	})
}
