/*
Command k8s-offline-gc is a quick&dirty way to delete orphaned objecets on k8s.

Usage:

    kubectl get secrets -o json >/tmp/secrets.json
    kubectl get jobs -o json >/tmp/jobs.json
    k8s-offline-gc /tmp/{secrets,jobs}.json | xargs -0 -I% sh -c 'kubectl %'
*/
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/juju/errors"
)

type list struct {
	Items []*resource `json:"items"`
}

type resource struct {
	Metadata   metadata `json:"metadata"`
	Kind       string   `json:"kind"`
	APIVersion string   `json:"apiVersion"`
}

type metadata struct {
	Name            string           `json:"name"`
	Namespace       string           `json:"namespace"`
	OwnerReferences []ownerReference `json:"ownerReferences"`
}

type ownerReference struct {
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
}

type key string

func (r *resource) key() key {
	return makeKey(r.Kind, r.APIVersion, r.Metadata.Namespace, r.Metadata.Name)
}

func (r *ownerReference) key(namespace string) key {
	return makeKey(r.Kind, r.APIVersion, namespace, r.Name)
}

func makeKey(kind, apiVersion, namespace, name string) key {
	return key(fmt.Sprintf("%s:%s:%s:%s", strings.ToLower(kind), apiVersion, namespace, name))
}

type store map[key]*resource

func newStore() store { return store{} }

func (s store) fillFromFile(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return errors.Trace(err)
	}
	defer f.Close()

	var l list
	if err := json.NewDecoder(f).Decode(&l); err != nil {
		return errors.Trace(err)
	}

	s.fillFromList(l)
	return nil
}

func (s store) fillFromList(l list) {
	for _, r := range l.Items {
		s[r.key()] = r
	}
}

// orphans returns a slice of resources that are owned by a resource not present in the store.
func (s store) orphans() list {
	res := list{Items: []*resource{}}

	for _, r := range s {
		if len(r.Metadata.OwnerReferences) == 1 {
			owner := r.Metadata.OwnerReferences[0]
			k := owner.key(r.Metadata.Namespace)

			if _, ok := s[k]; !ok {
				res.Items = append(res.Items, r)
			}
		}
	}
	return res
}

func run(files []string) error {
	s := newStore()
	for _, f := range files {
		if err := s.fillFromFile(f); err != nil {
			return errors.Trace(err)
		}
	}

	for _, r := range s.orphans().Items {
		fmt.Printf("-n %s delete %s %s\000", r.Metadata.Namespace, r.Kind, r.Metadata.Name)
	}
	return nil
}

func main() {
	flag.Parse()
	if err := run(flag.Args()); err != nil {
		log.Fatalf("%+v", err)
	}

}
