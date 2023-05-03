package main

import (
	"errors"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/KompiTech/rmap"
	"github.com/antchfx/xmlquery"
	"github.com/go-andiamo/splitter"
)

func isDir(dir string) bool {
	dirInfo, err := os.Stat(dir)
	if err != nil {
		log.Panic(err.Error())
	}

	return dirInfo.IsDir()
}

func existsFile(file string) bool {
	_, err := os.Stat(file)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false
		}
		log.Panic(err.Error())
	}
	return true
}

func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		log.Print("Use one or more directories with existing Goland projects as arguments")
		os.Exit(1)
	}

	for _, dir := range flag.Args() {

		if err := convertDir(dir); err != nil {
			log.Fatal(err.Error())
		}
	}
}

func getConfigurations(xmlFile string) (rmap.Rmap, error) {
	f, err := os.Open(xmlFile)
	if err != nil {
		return rmap.Rmap{}, err
	}

	defer func() { _ = f.Close() }()

	doc, err := xmlquery.Parse(f)
	if err != nil {
		return rmap.Rmap{}, err
	}

	configs := []rmap.Rmap{}

	for _, xmlC := range xmlquery.Find(doc, "//project/component[@name='RunManager']/configuration") {
		name := xmlC.SelectAttr("name")
		if name == "" {
			continue
		}

		// "name": "Launch Package",
		// "type": "go",
		// "request": "launch",
		// "mode": "auto",
		// "program": "${fileDirname}",
		// "args": ["/home/werk/Code/itsm-cgi"]

		filePath := xmlC.SelectElement("filePath")
		if filePath == nil {
			continue
		}

		log.Printf("Converting configuration: %s", name)

		config := rmap.NewFromMap(map[string]interface{}{
			"name":    name,
			"type":    "go",
			"request": "launch",
			"program": strings.Replace(filePath.SelectAttr("value"), "$PROJECT_DIR$/", "", 1),
		})

		if xmlParams := xmlC.SelectElement("parameters"); xmlParams != nil {
			log.Printf("Found exec arguments")
			params := xmlParams.SelectAttr("value")

			s, err := splitter.NewSplitter(' ', splitter.SingleQuotes)
			if err != nil {
				return rmap.Rmap{}, err
			}

			args, err := s.Split(params)
			if err != nil {
				return rmap.Rmap{}, err
			}

			config.Mapa["args"] = args
		}

		if xmlEnvs := xmlC.SelectElement("envs"); xmlEnvs != nil {
			log.Printf("Found ENV vars", xmlEnvs)

			envs := rmap.NewEmpty()
			envIter := xmlEnvs.SelectElements("env")

			for _, e := range envIter {
				envs.Mapa[e.SelectAttr("name")] = e.SelectAttr("value")
			}

			config.Mapa["env"] = envs
		}

		configs = append(configs, config)
	}

	out := rmap.NewFromMap(map[string]interface{}{
		"version":        "0.2.0",
		"configurations": configs,
	})

	return out, nil

}

func convertDir(dir string) error {
	if !isDir(dir) {
		log.Printf("Argument %s is not a directory, skipped", dir)
		return nil
	}

	xmlFile := filepath.Join(dir, ".idea/workspace.xml")

	if _, err := os.Stat(xmlFile); errors.Is(err, os.ErrNotExist) {
		log.Printf("File %s does not exist, skipped", xmlFile)
		return nil
	}

	log.Printf("Found goLand configuration %s", xmlFile)

	vsCodeDir := filepath.Join(dir, ".vscode/")

	if !isDir(vsCodeDir) {
		log.Printf("VSCode dir %s does not exist, creating", vsCodeDir)
		if err := os.Mkdir(vsCodeDir, 0644); err != nil {
			return err
		}
	}

	out, err := getConfigurations(xmlFile)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(vsCodeDir, "launch.json"), out.Bytes(), 0644)
}
