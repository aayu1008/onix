/*
  Onix ServerConfig Manager - Artie
  Copyright (c) 2018-2020 by www.gatblau.org
  Licensed under the Apache License, Version 2.0 at http://www.apache.org/licenses/LICENSE-2.0
  Contributors to this project, hereby assign copyright in this code to the project,
  to be licensed under the same terms as the rest of the code.
*/
package core

import (
	"fmt"
	"log"
	"os/user"
	"path/filepath"
)

const CliName = "artie"

// gets the user home directory
func HomeDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return usr.HomeDir
}

// gets the root path of the local registry
func RegistryPath() string {
	return filepath.Join(HomeDir(), fmt.Sprintf(".%s", CliName))
}
