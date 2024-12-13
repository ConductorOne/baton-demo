package main

import (
	"github.com/conductorone/baton-sdk/pkg/field"
)

var (
	dbFile = field.StringField("db-file", field.WithDescription("A file to which the database will be written ($BATON_DB_FILE)\nexample: /path/to/dbfile.db"))
	initDB = field.BoolField("init-db", field.WithDescription("Whether to initialize the database ($BATON_INIT_DB)\nexample: true"))
)

var relationships = []field.SchemaFieldRelationship{}

var configuration = field.NewConfiguration([]field.SchemaField{
	dbFile, initDB,
}, relationships...)
