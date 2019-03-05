#!/bin/bash
GO15VENDOREXPERIMENT=1 CGO_ENABLED=0 go build -a -ldflags '-s' -installsuffix cgo .
