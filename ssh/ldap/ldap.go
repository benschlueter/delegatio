/* SPDX-License-Identifier: AGPL-3.0-only
 * Copyright (c) Benedict Schlueter
 */

package ldap

import (
	"fmt"

	"github.com/benschlueter/delegatio/internal/config"
	goLdap "github.com/go-ldap/ldap/v3"
	"go.uber.org/zap"
)

type Ldap struct {
	log        *zap.Logger
	address    string
	dn         string
	attributes []string
}

func NewLdap(logger *zap.Logger) *Ldap {
	address := "ldaps://ldaps-rz-1.ethz.ch"
	dn := "cn=%s,ou=users,ou=nethz,ou=id,ou=auth,o=ethz,c=ch"
	// https://help.switch.ch/aai/support/documents/attributes/
	attributes := []string{
		"swissEduPersonMatriculationNumber",
		"swissEduPersonOrganizationalMail",
		"swissEduPersonGender",
		"eduPersonAffiliation",
		"surname",
		"givenName",
		"mail",
		"uid",
	}

	return &Ldap{
		address:    address,
		dn:         dn,
		attributes: attributes,
		log:        logger,
	}
}

func (l *Ldap) dial(username, password string) (*goLdap.Conn, error) {
	// Connect to LDAP server
	patchedDn := fmt.Sprintf(l.dn, username)
	connection, err := goLdap.DialURL(l.address)
	if err != nil {
		l.log.Error("ldap dial", zap.Error(err))
		return nil, err
	}
	// Bind to LDAP server
	l.log.Info("binding to ldap server", zap.String("dn", patchedDn))
	if err := connection.Bind(patchedDn, password); err != nil {
		l.log.Error("ldap bind", zap.Error(err), zap.String("dn", patchedDn))
		return nil, err
	}
	return connection, nil
}

func (l *Ldap) Search(username, password string) (*config.UserInformation, error) {
	l.log.Info("searching for user", zap.String("username", username))
	connection, err := l.dial(username, password)
	if err != nil {
		return nil, err
	}
	defer connection.Close()

	patchedDn := fmt.Sprintf(l.dn, username)

	searchRequest := goLdap.NewSearchRequest(
		patchedDn,
		goLdap.ScopeBaseObject,
		goLdap.NeverDerefAliases,
		0,
		0,
		false,
		fmt.Sprintf("(objectClass=*)"),
		l.attributes,
		nil,
	)
	sr, err := connection.Search(searchRequest)
	if err != nil {
		l.log.Error("ldap search", zap.Error(err), zap.String("dn", l.dn))
		return nil, err
	}
	// Sanity check: make sure the user is unique
	if len(sr.Entries) != 1 {
		l.log.Error("ldap user isn't unique", zap.String("dn", l.dn))
		return nil, fmt.Errorf("user not unique %s", patchedDn)
	}

	return &config.UserInformation{
		Username:   username,
		Uuid:       username + "-" + sr.Entries[0].GetAttributeValue("swissEduPersonMatriculationNumber"),
		LegiNumber: sr.Entries[0].GetAttributeValue("swissEduPersonMatriculationNumber"),
		Email:      sr.Entries[0].GetAttributeValue("swissEduPersonOrganizationalMail"),
		RealName:   sr.Entries[0].GetAttributeValue("givenName") + " " + sr.Entries[0].GetAttributeValue("surname"),
		Gender:     sr.Entries[0].GetAttributeValue("swissEduPersonGender"),
	}, nil
}
