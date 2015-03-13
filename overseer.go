package main

import (
	"os"
	"fmt"
	"net"
	"sort"
	"strings"
	"text/template"
	"path/filepath"

	"github.com/BurntSushi/toml"
)


var (

	resourcesDirName = "/etc/overseer/resources"
	templatesDirName = "/etc/overseer/templates"
	stateFileName     = "/var/overseer/state.toml"
	interval         = 60

	resources = make(map[string]map[*Resource]bool)
	state     = make(map[string]map[string]bool)

)

type ResourceConfig struct {
	Resource Resource `toml:"template"`
}

type Resource struct {
	Src   string
	Dest  string
	Hosts []string
	Uid int
	Gid int
	Mode string
	ReloadCmd string `toml:"reload_cmd"`
	Domain string
}


func main() {

	//load resources files and create index "domain name" -> "resource to generate" 
	resourcesDir, err := os.Open(resourcesDirName)
	defer func(){resourcesDir.Close()}()
	if err != nil { panic(err) }

	resourcesFiles, err := resourcesDir.Readdir(0)
	if err != nil { panic(err) }

	
	for _, resourceFile := range resourcesFiles {
		
		if filepath.Ext(resourceFile.Name()) != ".toml" ||  resourceFile.IsDir() {continue}
		
		var rc *ResourceConfig
		_, err := toml.DecodeFile(filepath.Join(resourcesDirName, resourceFile.Name()), &rc)
		if err != nil { panic(err) }

		for _, host := range rc.Resource.Hosts {
			host = strings.Join([]string{host, rc.Resource.Domain}, ".")
			if resources[host] == nil {resources[host]=make(map[*Resource]bool)}
			resources[host][&rc.Resource] = true
		}
	}

	fmt.Println(resources)

	//load state file
	_, err = toml.DecodeFile(stateFileName, &state)
	if err != nil && !os.IsNotExist(err) { panic(err) }
	

	resourcesToUpdate := make(map[*Resource]bool)
	newState := make(map[string]map[string]bool)
	//find ips to change
	for host, resourcesset := range resources {
		ips, err := net.LookupIP(host)
		if err != nil { panic(err) }
		newState[host] = make(map[string]bool)
		for _, ip := range ips {
			newState[host][ip.String()] = true
			if _, stateOk := state[host][ip.String()]; !stateOk{
				for resource, v := range resourcesset {
					resourcesToUpdate[resource] = v
				}
			}

		}

	}

	fmt.Println(resourcesToUpdate)
	fmt.Println(state)
	fmt.Println(newState)


	//make list of ips
	ips := make(map[string][]string)
	for host, ipsSet := range newState {
		ipsList := make([]string, 0, len(ipsSet))
		for ip, _ := range ipsSet {
			ipsList = append(ipsList, ip)
		}
		sort.Strings(ipsList)
		ips[host] = ipsList
	} 


	//generate resources
	for resource, _ := range resourcesToUpdate {
		tmpl, err := template.ParseFiles(filepath.Join(templatesDirName, resource.Src))
		if err != nil { panic(err) }
		destFile, err := os.Create(resource.Dest)
		defer func(){destFile.Close()}()
		if err != nil { panic(err) }
		err = tmpl.Execute(destFile, ips)
		if err != nil { panic(err) }
	}



	//write state file
	stateFile, err := os.Create(stateFileName)
	defer func(){stateFile.Close()}()
	if err != nil { panic(err) }
	err = toml.NewEncoder(stateFile).Encode(&newState)
	state = newState
	
}



// func main() {

// 	host := os.Args[1]
	
// 	ips, err := net.LookupIP(host)
// 	if err != nil { panic(err) }

// 	ipStrings := make([]string, len(ips))
// 	for index, value := range ips {
// 		ipStrings[index]=value.String()
// 	}
// 	sort.Strings(ipStrings)

// 	tmpl, err := template.ParseGlob("*.tmpl")
// 	if err != nil { panic(err) }

// 	err = tmpl.Execute(os.Stdout, ipStrings)
// 	if err != nil { panic(err) }
	
// }