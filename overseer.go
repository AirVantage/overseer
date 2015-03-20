package main

import (
	"os"
	"os/exec"
	"log"
	"net"
	"sort"
	"time"
	"strings"
	"text/template"
	"path/filepath"

	"github.com/BurntSushi/toml"
)


var (

	resourcesDirName = "/etc/overseer/resources"
	templatesDirName = "/etc/overseer/templates"
	stateFileName    = "/var/overseer/state.toml"
	interval         = 60

	resources = make(map[string]map[*Resource]bool)
	state     = make(map[string]map[string]bool)

)

type ResourceConfig struct {
	Resource Resource `toml:"template"`
}

type Resource struct {
	Src       string
	Dest      string
	Hosts     []string
	Uid       int
	Gid       int
	Mode      string
	ReloadCmd string `toml:"reload_cmd"`
	Domain    string
}

func init() {

	log.Println("Start init")

	// -----------------------------------------------------------------------------
	// load resources files and create index "domain name" -> "resource to generate"
	// -----------------------------------------------------------------------------

	log.Println("Load resources from", resourcesDirName)

	resourcesDir, err := os.Open(resourcesDirName)
	defer func(){resourcesDir.Close()}()
	if err != nil { log.Fatal(err) }

	resourcesFiles, err := resourcesDir.Readdir(0)
	if err != nil { log.Fatal(err) }

	for _, resourceFile := range resourcesFiles {
		
		if filepath.Ext(resourceFile.Name()) != ".toml" ||  resourceFile.IsDir() {continue}
		
		var rc *ResourceConfig
		_, err := toml.DecodeFile(filepath.Join(resourcesDirName, resourceFile.Name()), &rc)
		if err != nil { log.Fatal(err) }

		log.Println("Read File", resourceFile.Name(), ":", rc)

		for _, host := range rc.Resource.Hosts {
			host = strings.Join([]string{host, rc.Resource.Domain}, ".")
			if resources[host] == nil {resources[host]=make(map[*Resource]bool)}
			resources[host][&rc.Resource] = true
		}
	}

	// ---------------
	// load state file
	// ---------------

	// create file path if it does not exist

	err = os.MkdirAll(filepath.Dir(stateFileName), 0777)
	if err != nil { log.Fatal(err) }

	_, err = toml.DecodeFile(stateFileName, &state)
	if err != nil && !os.IsNotExist(err) { log.Fatal(err) }	

	log.Println("Load state from", stateFileName, ":", state)

	log.Println("Init done")

}

func iterate() {

	log.Println("Start iteration")

	log.Println("Find Resources to update")

	resourcesToUpdate := make(map[*Resource]bool)
	newState := make(map[string]map[string]bool)
	
	//find host ips to update
	for host, resourcesset := range resources {
		
		ips, err := net.LookupIP(host)
		if err != nil { log.Fatal(err) }
		newState[host] = make(map[string]bool)

		changed := false
		
		for _, ip := range ips {
			newState[host][ip.String()] = true
			if _, stateOk := state[host][ip.String()]; !stateOk {
				changed = true
				log.Println("For host", host, "new IP:", ip)
			}

		}

		for oldIp, _ := range state[host] {
			if _, stateOk := newState[host][oldIp]; !stateOk {
				changed = true
				log.Println("For host", host, "deprecated IP:", oldIp)
			}
		}

		if changed {
			for resource, v := range resourcesset {
				log.Println("For host", host, "update ressource:", resource)
				resourcesToUpdate[resource] = v
			}			
		}


	}


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

	log.Println("Update resources and restart processes")
	//generate resources
	for resource, _ := range resourcesToUpdate {
		
		tmpl, err := template.ParseFiles(filepath.Join(templatesDirName, resource.Src))
		if err != nil { log.Fatal(err) }
		err = os.MkdirAll(filepath.Dir(resource.Dest), 0777)
	    if err != nil { log.Fatal(err) }
		destFile, err := os.Create(resource.Dest)
		defer func(){destFile.Close()}()
		if err != nil { log.Fatal(err) }
		err = tmpl.Execute(destFile, ips)
		if err != nil { log.Fatal(err) }
		log.Println("For resource", resource, "update file", resource.Dest)

		if resource.ReloadCmd == "" {continue}

		//cmdSplit := strings.Fields(resource.ReloadCmd)
		//cmd := exec.Command(cmdSplit[0], cmdSplit[1:]...)
		cmd := exec.Command("bash", "-c", resource.ReloadCmd)
		log.Println(cmd)
	    err = cmd.Start()
	    if err != nil {
	    	log.Fatal(err)
	    }
	    log.Println("For resource", resource, "start reload cmd", resource.ReloadCmd)
	    err = cmd.Wait()
	    if err != nil {
	    	log.Println("For resource", resource, "reload cmd", resource.ReloadCmd, "finished with error", err)
	    } else {
	    	log.Println("For resource", resource, "reload cmd", resource.ReloadCmd, "finished successfuly")
	    }
	    

	}

	//write state file
	stateFile, err := os.Create(stateFileName)
	defer func(){stateFile.Close()}()
	if err != nil { log.Fatal(err) }
	err = toml.NewEncoder(stateFile).Encode(&newState)
	state = newState
	log.Println("Log state", state, "in file", state)

	log.Println("Iteration done")
	
}

func main(){
	for {
		iterate()
		time.Sleep(time.Duration(interval)*time.Second)
	}
}
