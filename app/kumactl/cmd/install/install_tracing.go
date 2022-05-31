// Deprecated: `kumactl install tracing` is deprecated, use `kumactl install observability` instead

package install

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	kumactl_data "github.com/kumahq/kuma/app/kumactl/data"
	kumactl_cmd "github.com/kumahq/kuma/app/kumactl/pkg/cmd"
	"github.com/kumahq/kuma/app/kumactl/pkg/install/data"
	"github.com/kumahq/kuma/app/kumactl/pkg/install/k8s"
)

type tracingTemplateArgs struct {
	Namespace string
}

func newInstallTracing(pctx *kumactl_cmd.RootContext) *cobra.Command {
	args := pctx.InstallTracingContext.TemplateArgs
	cmd := &cobra.Command{
		Use:        "tracing",
		Short:      "Install Tracing backend in Kubernetes cluster (Jaeger)",
		Long:       `Install Tracing backend in Kubernetes cluster (Jaeger) in its own namespace.`,
		Deprecated: "We're migrating to `observability`, please use `install observability`",
		RunE: func(cmd *cobra.Command, _ []string) error {
			templateArgs := tracingTemplateArgs{
				Namespace: args.Namespace,
			}

			templateFiles, err := data.ReadFiles(kumactl_data.InstallDeprecatedTracingFS())
			if err != nil {
				return errors.Wrap(err, "Failed to read template files")
			}

			renderedFiles, err := renderFiles(templateFiles, templateArgs, simpleTemplateRenderer)
			if err != nil {
				return errors.Wrap(err, "Failed to render template files")
			}

			sortedResources, err := k8s.SortResourcesByKind(renderedFiles)
			if err != nil {
				return errors.Wrap(err, "Failed to sort resources by kind")
			}

			singleFile := data.JoinYAML(sortedResources)

			if _, err := cmd.OutOrStdout().Write(singleFile.Data); err != nil {
				return errors.Wrap(err, "Failed to output rendered resources")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&args.Namespace, "namespace", args.Namespace, "namespace to install tracing to")
	cmd.Print("# ") // HACK: by default cobra for deprecated commands will output a warning here:
	// https://github.com/spf13/cobra/blob/5b11656e45a6a6579298a3b28c71f456ff196ad6/command.go#L785
	// so this adds '#' to the generated output so we don't fail on this:
	// kumactl install tracing | kubectl apply -f -
	return cmd
}
