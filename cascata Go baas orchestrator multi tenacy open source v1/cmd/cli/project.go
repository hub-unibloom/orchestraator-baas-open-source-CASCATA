package main

import (
	"flag"
	"fmt"
	"os"
)

// handleProject routes commands targeting 'cascata project <subcommand>'
func handleProject(cfg CommandConfig, args []string) {
	if len(args) == 0 {
		Fatal(cfg, "MISSING_COMMAND", "Subcommand missing for project", "Valid options: list, create, export")
	}

	sub := args[0]
	switch sub {
	case "list":
		projectList(cfg, args[1:])
	case "create":
		projectCreate(cfg, args[1:])
	case "export":
		projectExport(cfg, args[1:])
	default:
		Fatal(cfg, "UNKNOWN_COMMAND", "Unknown project command", sub)
	}
}

func projectList(cfg CommandConfig, args []string) {
	client := NewAPIClient(cfg)
	var projects []interface{}
	
	err := client.Do("GET", "/projects", nil, &projects)
	if err != nil {
		Fatal(cfg, "API_ERROR", "Failed to fetch projects", err.Error())
	}

	PrintSuccess(cfg, projects, fmt.Sprintf("Found %d projects.", len(projects)))
}

func projectCreate(cfg CommandConfig, args []string) {
	fs := flag.NewFlagSet("project-create", flag.ExitOnError)
	name := fs.String("name", "", "Display name of the project")
	slug := fs.String("slug", "", "Unique project slug")
	fs.Parse(args)

	if *name == "" || *slug == "" {
		Fatal(cfg, "MISSING_FLAG", "Must provide --name and --slug", "")
	}

	payload := map[string]string{
		"name": *name,
		"slug": *slug,
	}

	client := NewAPIClient(cfg)
	var result interface{}
	err := client.Do("POST", "/projects", payload, &result)
	if err != nil {
		Fatal(cfg, "API_ERROR", "Failed to birth project", err.Error())
	}

	PrintSuccess(cfg, result, fmt.Sprintf("Project '%s' birthed successfully.", *slug))
}

func projectExport(cfg CommandConfig, args []string) {
	fs := flag.NewFlagSet("project-export", flag.ExitOnError)
	slug := fs.String("slug", "", "Project slug to export as .caf")
	output := fs.String("output", "", "Output filename (defaults to <slug>.caf)")
	fs.Parse(args)

	if *slug == "" { Fatal(cfg, "MISSING_FLAG", "Must provide --slug", "") }
	if *output == "" { *output = *slug + ".caf" }

	client := NewAPIClient(cfg)
	
	f, err := os.Create(*output)
	if err != nil {
		Fatal(cfg, "FS_ERROR", "Failed to create output file", err.Error())
	}
	defer f.Close()

	fmt.Printf("Exporting project %s to %s...\n", *slug, *output)
	err = client.Download(fmt.Sprintf("/projects/export?slug=%s", *slug), f)
	if err != nil {
		Fatal(cfg, "API_ERROR", "Export failed", err.Error())
	}

	fmt.Printf("\nExport completed: %s\n", *output)
}
