// Copyright 2017 syzkaller project authors. All rights reserved.
// Use of this source code is governed by Apache 2 LICENSE that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/syzkaller/pkg/db"
	"github.com/google/syzkaller/pkg/hash"
	"github.com/google/syzkaller/pkg/osutil"
)

func main() {
	var (
		flagVersion = flag.Uint64("version", 0, "database version")
	)
	flag.Parse()
	args := flag.Args()
	if len(args) != 3 {
		usage()
	}
	switch args[0] {
	case "pack":
		pack(args[1], args[2], *flagVersion)
	case "unpack":
		unpack(args[1], args[2])
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage:\n")
	fmt.Fprintf(os.Stderr, "  syz-db pack dir corpus.db\n")
	fmt.Fprintf(os.Stderr, "  syz-db unpack corpus.db dir\n")
	os.Exit(1)
}

func pack(dir, file string, version uint64) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		failf("failed to read dir: %v", err)
	}
	os.Remove(file)
	db, err := db.Open(file)
	if err != nil {
		failf("failed to open database file: %v", err)
	}
	if err := db.BumpVersion(version); err != nil {
		failf("failed to bump database version: %v", err)
	}
	for _, file := range files {
		data, err := ioutil.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			failf("failed to read file %v: %v", file.Name(), err)
		}
		var seq uint64
		key := file.Name()
		if parts := strings.Split(file.Name(), "-"); len(parts) == 2 {
			var err error

			if seq, err = strconv.ParseUint(parts[1], 10, 64); err == nil {
				key = parts[0]
			}
		}
		if sig := hash.String(data); key != sig {
			fmt.Fprintf(os.Stderr, "fixing hash %v -> %v\n", key, sig)
			key = sig
		}
		db.Save(key, data, seq)
	}
	if err := db.Flush(); err != nil {
		failf("failed to save database file: %v", err)
	}
}

func unpack(file, dir string) {
	db, err := db.Open(file)
	if err != nil {
		failf("failed to open database: %v", err)
	}
	osutil.MkdirAll(dir)
	for key, rec := range db.Records {
		fname := filepath.Join(dir, key)
		if rec.Seq != 0 {
			fname += fmt.Sprintf("-%v", rec.Seq)
		}
		if err := osutil.WriteFile(fname, rec.Val); err != nil {
			failf("failed to output file: %v", err)
		}
	}
}

func failf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}
