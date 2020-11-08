/*
  Onix Config Manager - Artie
  Copyright (c) 2018-2020 by www.gatblau.org
  Licensed under the Apache License, Version 2.0 at http://www.apache.org/licenses/LICENSE-2.0
  Contributors to this project, hereby assign copyright in this code to the project,
  to be licensed under the same terms as the rest of the code.
*/
package registry

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gatblau/onix/artie/core"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"
)

// the default local registry implemented as a file system
type FileRegistry struct {
	Repositories []*repository `json:"repositories"`
}

// find the repository specified by name
func (r *FileRegistry) findRepository(name *core.ArtieName) *repository {
	// find repository using artefact name
	for _, repository := range r.Repositories {
		if repository.Repository == name.FullyQualifiedName() {
			return repository
		}
	}
	// find repository using artefact Id
	for _, repository := range r.Repositories {
		for _, artie := range repository.Artefacts {
			if strings.Contains(artie.Id, name.Name) {
				return repository
			}
		}
	}
	return nil
}

// return all the artefacts within the same repository
func (r *FileRegistry) GetArtefactsByName(name *core.ArtieName) ([]*artefact, bool) {
	var artefacts = make([]*artefact, 0)
	for _, repository := range r.Repositories {
		if repository.Repository == name.FullyQualifiedName() {
			for _, artefact := range repository.Artefacts {
				artefacts = append(artefacts, artefact)
			}
			break
		}
	}
	if len(artefacts) > 0 {
		return artefacts, true
	}
	return nil, false
}

// return the artefact that matches the specified:
// - domain/group/name:tag or
// - artefact id substring or
// nil if not found in the FileRegistry
func (r *FileRegistry) GetArtefact(name *core.ArtieName) *artefact {
	// go through the artefacts in the repository and check for Id matches
	artefactsFound := make([]*artefact, 0)

	// first gets the repository the artefact is in
	for _, repository := range r.Repositories {
		if repository.Repository == name.FullyQualifiedName() {
			// try and get it by id first
			for _, artefact := range repository.Artefacts {
				for _, tag := range artefact.Tags {
					// try and match against the full URI
					if tag == name.Tag {
						return artefact
					}
				}
			}
			break
		}
		for _, artefact := range repository.Artefacts {
			// try and match against the artefact ID substring
			if strings.Contains(artefact.Id, name.Name) {
				artefactsFound = append(artefactsFound, artefact)
			}
		}
		if len(artefactsFound) > 1 {
			core.RaiseErr("artefact Id hint provided is not sufficiently long to pin point the artifact, %d were found", len(artefactsFound))
		}
	}
	if len(artefactsFound) == 0 {
		return nil
	}
	return artefactsFound[0]
}

type repository struct {
	// the artefact repository (name without without tag)
	Repository string `json:"repository"`
	// the reference name of the artefact corresponding to different builds
	Artefacts []*artefact `json:"artefacts"`
}

type artefact struct {
	// a unique identifier for the artefact calculated as the checksum of the complete seal
	Id string `json:"id"`
	// the type of application in the artefact
	Type string `json:"type"`
	// the artefact actual file name
	FileRef string `json:"file_ref"`
	// the list of Tags associated with the artefact
	Tags []string `json:"tags"`
	// the size
	Size string `json:"size"`
	// the creation time
	Created string `json:"created"`
}

func (a artefact) HasTag(tag string) bool {
	for _, t := range a.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

// create a localRepo management structure
func NewFileRegistry() *FileRegistry {
	r := &FileRegistry{
		Repositories: []*repository{},
	}
	// load local registry
	r.load()
	return r
}

// the local Path to the local FileRegistry
func (r *FileRegistry) Path() string {
	return core.RegistryPath()
}

// return the FileRegistry full file name
func (r *FileRegistry) file() string {
	return filepath.Join(r.Path(), "repository.json")
}

// save the state of the FileRegistry
func (r *FileRegistry) save() {
	core.Msg("updating local registry metadata")
	regBytes := core.ToJsonBytes(r)
	core.CheckErr(ioutil.WriteFile(r.file(), regBytes, os.ModePerm), "fail to update local registry metadata")
}

// load the content of the FileRegistry
func (r *FileRegistry) load() {
	// check if localRepo file exist
	_, err := os.Stat(r.file())
	if err != nil {
		// then assume localRepo.json is not there: try and create it
		r.save()
	} else {
		regBytes, err := ioutil.ReadFile(r.file())
		if err != nil {
			log.Fatal(err)
		}
		err = json.Unmarshal(regBytes, r)
		if err != nil {
			log.Fatal(err)
		}
	}
}

// Add the artefact and seal to the FileRegistry
func (r *FileRegistry) Add(filename string, name *core.ArtieName, s *core.Seal) {
	core.Msg("adding artefact to local registry: %s", name)
	// gets the full base name (with extension)
	basename := filepath.Base(filename)
	// gets the basename directory only
	basenameDir := filepath.Dir(filename)
	// gets the base name extension
	basenameExt := path.Ext(basename)
	// gets the base name without extension
	basenameNoExt := strings.TrimSuffix(basename, path.Ext(basename))
	// if the file to add is not a zip file
	if basenameExt != ".zip" {
		log.Fatal(errors.New(fmt.Sprintf("the localRepo can only accept zip files, the extension provided was %s", basenameExt)))
	}
	// move the zip file to the localRepo folder
	core.CheckErr(RenameFile(filename, filepath.Join(r.Path(), basename), false), "failed to move artefact zip file to the local registry")
	// now move the seal file to the localRepo folder
	core.CheckErr(RenameFile(filepath.Join(basenameDir, fmt.Sprintf("%s.json", basenameNoExt)), filepath.Join(r.Path(), fmt.Sprintf("%s.json", basenameNoExt)), false), "failed to move artefact seal file to the local registry")
	// untag artefact artefact (if any)
	r.unTag(name, name.Tag)
	// find the repository
	repo := r.findRepository(name)
	// if the repo does not exist the creates it
	if repo == nil {
		repo = &repository{
			Repository: name.FullyQualifiedName(),
			Artefacts:  make([]*artefact, 0),
		}
		r.Repositories = append(r.Repositories, repo)
	}
	// creates a new artefact
	artefacts := append(repo.Artefacts, &artefact{
		Id:      core.ArtefactId(s),
		Type:    s.Manifest.Type,
		FileRef: basenameNoExt,
		Tags:    []string{name.Tag},
		Size:    s.Manifest.Size,
		Created: s.Manifest.Time,
	})
	repo.Artefacts = artefacts
	// persist the changes
	r.save()
}

func (r *FileRegistry) removeArtefactById(a []*artefact, id string) []*artefact {
	i := -1
	// find the value to remove
	for ix := 0; ix < len(a); ix++ {
		if strings.Contains(a[ix].Id, id) {
			i = ix
			break
		}
	}
	if i == -1 {
		return a
	}
	// Remove the element at index i from a.
	a[i] = a[len(a)-1] // Copy last element to index i.
	a[len(a)-1] = nil  // Erase last element (write zero value).
	a = a[:len(a)-1]   // Truncate slice.
	return a
}

func (r *FileRegistry) removeRepoByName(a []*repository, name *core.ArtieName) []*repository {
	i := -1
	// find an artefact with the specified tag
	for ix := 0; ix < len(a); ix++ {
		if a[ix].Repository == name.FullyQualifiedName() {
			i = ix
			break
		}
	}
	if i == -1 {
		return a
	}
	// Remove the element at index i from a.
	a[i] = a[len(a)-1] // Copy last element to index i.
	a[len(a)-1] = nil  // Erase last element (write zero value).
	a = a[:len(a)-1]   // Truncate slice.
	return a
}

// remove a given tag from an artefact
func (r *FileRegistry) unTag(name *core.ArtieName, tag string) {
	artie := r.GetArtefact(name)
	if artie != nil {
		core.Msg("untagging %s", name)
		artie.Tags = core.RemoveElement(artie.Tags, tag)
	}
}

// remove a given tag from an artefact
func (r *FileRegistry) Tag(sourceName *core.ArtieName, targetName *core.ArtieName) {
	sourceArtie := r.GetArtefact(sourceName)
	if sourceArtie == nil {
		core.RaiseErr("source artefact %s does not exit", sourceName)
	}
	if targetName.IsInTheSameRepositoryAs(sourceName) {
		if !sourceArtie.HasTag(targetName.Tag) {
			core.Msg("tagging %s", sourceName)
			sourceArtie.Tags = append(sourceArtie.Tags, targetName.Tag)
			r.save()
			return
		} else {
			core.Msg("already tagged")
			return
		}
	} else {
		targetRepository := r.findRepository(targetName)
		newArtie := *sourceArtie
		// if the target artefact repository does not exist then create it
		if targetRepository == nil {
			core.Msg("tagging %s", sourceName)
			newArtie.Tags = []string{targetName.Tag}
			r.Repositories = append(r.Repositories, &repository{
				Repository: targetName.FullyQualifiedName(),
				Artefacts: []*artefact{
					&artefact{
						Id:      sourceArtie.Id,
						Type:    sourceArtie.Type,
						FileRef: sourceArtie.FileRef,
						Tags:    []string{targetName.Tag},
						Size:    sourceArtie.Size,
						Created: sourceArtie.Created,
					},
				},
			})
			r.save()
			return
		} else {
			targetArtie := r.GetArtefact(targetName)
			// if the artefact exists in the repository
			if targetArtie != nil {
				// check if the tag already exists
				for _, tag := range targetArtie.Tags {
					if tag == targetName.Tag {
						core.Msg("already tagged")
					} else {
						// add the tag to the existing artefact
						targetArtie.Tags = append(targetArtie.Tags, targetName.Tag)
					}
				}
			} else {
				// check that an artefact with the Id of the source exists
				for _, a := range targetRepository.Artefacts {
					// if the target repository already contains the artefact Id
					if a.Id == sourceArtie.Id {
						// add a tag
						a.Tags = append(a.Tags, targetName.Tag)
						r.save()
						return
					}
				}
				// add a new artefact metadata in the existing repository
				targetRepository.Artefacts = append(targetRepository.Artefacts,
					&artefact{
						Id:      sourceArtie.Id,
						Type:    sourceArtie.Type,
						FileRef: sourceArtie.FileRef,
						Tags:    []string{targetName.Tag},
						Size:    sourceArtie.Size,
						Created: sourceArtie.Created,
					})
				r.save()
				return
			}
		}
	}
}

// remove all tags from the specified artefact
func (r *FileRegistry) unTagAll(name *core.ArtieName) {
	if artefs, exists := r.GetArtefactsByName(name); exists {
		// then it has to untag it, leaving a dangling artefact
		for _, artef := range artefs {
			for _, tag := range artef.Tags {
				artef.Tags = core.RemoveElement(artef.Tags, tag)
			}
		}
	}
	// persist changes
	r.save()
}

// List artefacts to stdout
func (r *FileRegistry) List() {
	// get a table writer for the stdout
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 12, ' ', 0)
	// print the header row
	_, err := fmt.Fprintln(w, "REPOSITORY\tTAG\tARTEFACT ID\tARTEFACT TYPE\tCREATED\tSIZE")
	core.CheckErr(err, "failed to write table header")
	// repository, tag, artefact id, created, size
	for _, repo := range r.Repositories {
		for _, a := range repo.Artefacts {
			// if the artefact is dangling (no tags)
			if len(a.Tags) == 0 {
				_, err := fmt.Fprintln(w, fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s",
					repo.Repository,
					"<none>",
					a.Id[7:19],
					a.Type,
					toElapsedLabel(a.Created),
					a.Size),
				)
				core.CheckErr(err, "failed to write output")
			}
			for _, tag := range a.Tags {
				_, err := fmt.Fprintln(w, fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s",
					repo.Repository,
					tag,
					a.Id[7:19],
					a.Type,
					toElapsedLabel(a.Created),
					a.Size),
				)
				core.CheckErr(err, "failed to write output")
			}
		}
	}
	err = w.Flush()
	core.CheckErr(err, "failed to flush output")
}

// list (quiet) artefact IDs only
func (r *FileRegistry) ListQ() {
	// get a table writer for the stdout
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 10, ' ', 0)
	// repository, tag, artefact id, created, size
	for _, repo := range r.Repositories {
		for _, a := range repo.Artefacts {
			_, err := fmt.Fprintln(w, fmt.Sprintf("%s", a.Id[7:19]))
			core.CheckErr(err, "failed to write artefact Id")
		}
	}
	err := w.Flush()
	core.CheckErr(err, "failed to flush output")
}

func (r *FileRegistry) Push(name *core.ArtieName, remote Remote, credentials string) {
	// fetch the artefact info from the local registry
	artie := r.GetArtefact(name)
	if artie == nil {
		log.Fatal(errors.New(fmt.Sprintf("artefact %s not found in the local registry", name)))
	}
	// set up an http client
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	// execute the upload
	err := remote.UploadArtefact(client, name, r.Path(), artie.FileRef, credentials)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("pushed %s\n", name.String())
}

func (r *FileRegistry) Remove(names []*core.ArtieName) {
	for _, name := range names {
		// try and get the artefact by complete URI or id ref
		artie := r.GetArtefact(name)
		if artie == nil {
			fmt.Printf("name %s not found\n", name.Name)
			continue
		}
		// try to remove it using full name
		// remove the specified tag
		length := len(artie.Tags)
		r.unTag(name, name.Tag)
		// if the tag was successfully deleted
		if len(artie.Tags) < length {
			// if there are no tags left at the end then remove the repository
			if len(artie.Tags) == 0 {
				r.Repositories = r.removeRepoByName(r.Repositories, name)
				// only remove the files if there are no other repositories containing the same artefact!
				found := false
			Loop:
				for _, repo := range r.Repositories {
					for _, art := range repo.Artefacts {
						if art.Id == artie.Id {
							found = true
							break Loop
						}
					}
				}
				// no other repo contains the artefact so safe to remove the files
				if !found {
					r.removeFiles(artie)
				}
			}
			// persist changes
			r.save()
			log.Print(artie.Id)
		} else {
			// attempt to remove by Id (stored in the Name)
			repo := r.findRepository(name)
			repo.Artefacts = r.removeArtefactById(repo.Artefacts, name.Name)
			r.removeFiles(artie)
			r.save()
			log.Print(artie.Id)
		}
	}
}

// remove the files associated with an artefact
func (r *FileRegistry) removeFiles(artie *artefact) {
	// remove the zip file
	err := os.Remove(fmt.Sprintf("%s/%s.zip", r.Path(), artie.FileRef))
	if err != nil {
		log.Fatal(err)
	}
	// remove the json file
	err = os.Remove(fmt.Sprintf("%s/%s.json", r.Path(), artie.FileRef))
	if err != nil {
		log.Fatal(err)
	}
}

func (r *FileRegistry) Pull(name *core.ArtieName, remote Remote) {
}

// returns the elapsed time until now in human friendly format
func toElapsedLabel(rfc850time string) string {
	created, err := time.Parse(time.RFC850, rfc850time)
	if err != nil {
		log.Fatal(err)
	}
	elapsed := time.Since(created)
	seconds := elapsed.Seconds()
	minutes := elapsed.Minutes()
	hours := elapsed.Hours()
	days := hours / 24
	weeks := days / 7
	months := weeks / 4
	years := months / 12

	if math.Trunc(years) > 0 {
		return fmt.Sprintf("%d %s ago", int64(years), plural(int64(years), "year"))
	} else if math.Trunc(months) > 0 {
		return fmt.Sprintf("%d %s ago", int64(months), plural(int64(months), "month"))
	} else if math.Trunc(weeks) > 0 {
		return fmt.Sprintf("%d %s ago", int64(weeks), plural(int64(weeks), "week"))
	} else if math.Trunc(days) > 0 {
		return fmt.Sprintf("%d %s ago", int64(days), plural(int64(days), "day"))
	} else if math.Trunc(hours) > 0 {
		return fmt.Sprintf("%d %s ago", int64(hours), plural(int64(hours), "hour"))
	} else if math.Trunc(minutes) > 0 {
		return fmt.Sprintf("%d %s ago", int64(minutes), plural(int64(minutes), "minute"))
	}
	return fmt.Sprintf("%d %s ago", int64(seconds), plural(int64(seconds), "second"))
}

// turn label into plural if value is greater than one
func plural(value int64, label string) string {
	if value > 1 {
		return fmt.Sprintf("%ss", label)
	}
	return label
}

// the fully qualified name of the json Seal file in the local localReg
func (r *FileRegistry) regDirJsonFilename(uniqueIdName string) string {
	return fmt.Sprintf("%s/%s.json", r.Path(), uniqueIdName)
}

// the fully qualified name of the zip file in the local localReg
func (r *FileRegistry) regDirZipFilename(uniqueIdName string) string {
	return fmt.Sprintf("%s/%s.zip", r.Path(), uniqueIdName)
}