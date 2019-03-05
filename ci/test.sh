#!/bin/bash
GO15VENDOREXPERIMENT=1 go test $(go list ./... | grep -v /vendor/)
