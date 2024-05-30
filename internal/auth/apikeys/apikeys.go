package apikeys

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/llm-operator/cli/internal/auth/org"
	"github.com/llm-operator/cli/internal/auth/project"
	ihttp "github.com/llm-operator/cli/internal/http"
	"github.com/llm-operator/cli/internal/runtime"
	"github.com/llm-operator/cli/internal/ui"
	uv1 "github.com/llm-operator/user-manager/api/v1"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
)

const (
	pathPattern = "/organizations/%s/projects/%s/api-keys"
)

// Cmd is the root command for apikeys.
func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "api-keys",
		Short:              "API Keys commands",
		Args:               cobra.NoArgs,
		DisableFlagParsing: true,
	}
	cmd.AddCommand(createCmd())
	cmd.AddCommand(listCmd())
	cmd.AddCommand(deleteCmd())
	return cmd
}

func createCmd() *cobra.Command {
	var (
		name         string
		orgTitle     string
		projectTitle string
	)
	cmd := &cobra.Command{
		Use:  "create",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return create(cmd.Context(), name, orgTitle, projectTitle)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Name of the API key")
	cmd.Flags().StringVarP(&orgTitle, "organization-title", "o", "", "Title of the organization. The organization in the current context is used if not specified.")
	cmd.Flags().StringVarP(&projectTitle, "project-title", "p", "", "Title of the project. The project in the current context is used if not specified.")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func listCmd() *cobra.Command {
	var (
		orgTitle     string
		projectTitle string
	)
	cmd := &cobra.Command{
		Use:  "list",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return list(cmd.Context(), orgTitle, projectTitle)
		},
	}
	cmd.Flags().StringVarP(&orgTitle, "organization-title", "o", "", "Title of the organization. The organization in the current context is used if not specified.")
	cmd.Flags().StringVarP(&projectTitle, "project-title", "p", "", "Title of the project. The project in the current context is used if not specified.")
	return cmd
}

func deleteCmd() *cobra.Command {
	var (
		name         string
		orgTitle     string
		projectTitle string
	)
	cmd := &cobra.Command{
		Use:  "delete",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return delete(cmd.Context(), name, orgTitle, projectTitle)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Name of the API key")
	cmd.Flags().StringVarP(&orgTitle, "organization-title", "o", "", "Title of the organization. The organization in the current context is used if not specified.")
	cmd.Flags().StringVarP(&projectTitle, "project-title", "p", "", "Title of the project. The project in the current context is used if not specified.")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func create(ctx context.Context, name, orgTitle, projectTitle string) error {
	env, err := runtime.NewEnv(ctx)
	if err != nil {
		return err
	}

	org, project, err := findOrgAndProject(env, orgTitle, projectTitle)
	if err != nil {
		return err
	}

	req := &uv1.CreateAPIKeyRequest{
		Name:           name,
		OrganizationId: org.Id,
		ProjectId:      project.Id,
	}
	var resp uv1.APIKey
	path := fmt.Sprintf(pathPattern, org.Id, project.Id)
	if err := ihttp.NewClient(env).Send(http.MethodPost, path, &req, &resp); err != nil {
		return err
	}

	fmt.Printf("Created a new API key. Secret: %s\n", resp.Secret)
	return nil
}

func list(ctx context.Context, orgTitle, projectTitle string) error {
	env, err := runtime.NewEnv(ctx)
	if err != nil {
		return err
	}

	org, project, err := findOrgAndProject(env, orgTitle, projectTitle)
	if err != nil {
		return err
	}

	req := &uv1.ListAPIKeysRequest{
		OrganizationId: org.Id,
		ProjectId:      project.Id,
	}
	var resp uv1.ListAPIKeysResponse
	path := fmt.Sprintf(pathPattern, org.Id, project.Id)
	if err := ihttp.NewClient(env).Send(http.MethodGet, path, req, &resp); err != nil {
		return err
	}

	tbl := table.New("Name", "Owner", "Created At")
	ui.FormatTable(tbl)

	for _, k := range resp.Data {
		tbl.AddRow(k.Name, k.User.Id, time.Unix(k.CreatedAt, 0).Format(time.RFC3339))
	}

	tbl.Print()

	return nil
}

func delete(ctx context.Context, name, orgTitle, projectTitle string) error {
	env, err := runtime.NewEnv(ctx)
	if err != nil {
		return err
	}

	org, project, err := findOrgAndProject(env, orgTitle, projectTitle)
	if err != nil {
		return err
	}

	key, found, err := findKeyByName(ctx, env, name, org, project)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("API key %q not found", name)
	}

	req := &uv1.DeleteAPIKeyRequest{
		Id:             key.Id,
		OrganizationId: org.Id,
		ProjectId:      project.Id,
	}
	var resp uv1.DeleteAPIKeyResponse
	path := fmt.Sprintf(pathPattern, org.Id, project.Id)
	if err := ihttp.NewClient(env).Send(http.MethodDelete, fmt.Sprintf("%s/%s", path, key.Id), &req, &resp); err != nil {
		return err
	}

	fmt.Printf("Deleted the API key (ID: %q).\n", key.Id)

	return nil
}

func findOrgAndProject(env *runtime.Env, orgTitle, projectTitle string) (*uv1.Organization, *uv1.Project, error) {
	org, found, err := org.FindOrgByTitle(env, orgTitle)
	if err != nil {
		return nil, nil, err
	}
	if !found {
		return nil, nil, fmt.Errorf("organization not found")
	}

	project, found, err := project.FindProjectByTitle(env, projectTitle, orgTitle)
	if err != nil {
		return nil, nil, err
	}
	if !found {
		return nil, nil, fmt.Errorf("project not found in organization")
	}

	return org, project, nil
}

func findKeyByName(
	ctx context.Context,
	env *runtime.Env,
	name string,
	org *uv1.Organization,
	project *uv1.Project,
) (*uv1.APIKey, bool, error) {
	req := &uv1.ListAPIKeysRequest{
		OrganizationId: org.Id,
		ProjectId:      project.Id,
	}
	var resp uv1.ListAPIKeysResponse
	path := fmt.Sprintf(pathPattern, org.Id, project.Id)
	if err := ihttp.NewClient(env).Send(http.MethodGet, path, &req, &resp); err != nil {
		return nil, false, err
	}

	for _, k := range resp.Data {
		if k.Name == name {
			return k, true, nil
		}
	}
	return nil, false, nil
}
