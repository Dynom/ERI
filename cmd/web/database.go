package main

import (
	"database/sql"

	"github.com/sirupsen/logrus"

	"github.com/Dynom/ERI/validator/validations"

	"github.com/Dynom/ERI/cmd/web/hitlist"
)

type hitListRow struct {
	Domain      string `sql:"domain"`
	Recipient   string `sql:"recipient"`
	Validations int64  `sql:"validations"`
}

func preloadValues(conn *sql.DB, list *hitlist.HitList, logger logrus.FieldLogger) (uint, error) {

	if err := conn.Ping(); err != nil {
		return 0, err
	}

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

		logger.WithField("row", row).Info("Got one!")
		var err error
		if row.Recipient == "" {
			err = list.AddDomain(row.Domain, validations.Validations(row.Validations))
		} else {
			err = list.AddEmailAddress(row.Recipient+`@`+row.Domain, validations.Validations(row.Validations))
		}

		if err != nil {
			logger.WithError(err).Error("Error Adding e-mail address / domain to hit list")
			continue
		}

		dbCollected++
	}

	return dbCollected, nil
}

func registerPersistCallback(conn *sql.DB, list *hitlist.HitList, logger logrus.FieldLogger) {

	list.RegisterOnChange(func(r hitlist.Recipient, d string, v validations.Validations, c hitlist.ChangeType) {

		if c != hitlist.ChangeAdd {
			return
		}

		stmt, err := conn.Prepare(`
			INSERT INTO hitlist (domain, recipient, validations)
			VALUES ($1, $2, $3)
			ON CONFLICT (domain, recipient) DO UPDATE
			SET validations = EXCLUDED.validations`)

		if err != nil {
			panic(err)
		}

		_, err = stmt.Exec(d, r.String(), v)
		if err != nil {
			logger.WithError(err).Error("Couldn't persist change")
		} else {
			logger.WithFields(logrus.Fields{
				"domain":      d,
				"recipient":   r.String(),
				"validations": v.String(),
			}).Debug("Persisted (new) recipient")
		}
	})
}
