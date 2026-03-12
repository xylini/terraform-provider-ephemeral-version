package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var _ provider.Provider = &EphemeralVersionProvider{}

type EphemeralVersionProvider struct{}

func New() provider.Provider {
	return &EphemeralVersionProvider{}
}

func (p *EphemeralVersionProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ephemeralversion"
}

func (p *EphemeralVersionProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{}
}

func (p *EphemeralVersionProvider) Configure(_ context.Context, _ provider.ConfigureRequest, _ *provider.ConfigureResponse) {
}

func (p *EphemeralVersionProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewEphemeralVersionResource,
		NewEphemeralVersionMapResource,
	}
}

func (p *EphemeralVersionProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}
