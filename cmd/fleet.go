package cmd

import (
	"fmt"
	"os"
	"sync"
	"text/tabwriter"

	"github.com/melih-ucgun/veto/internal/config"
	"github.com/melih-ucgun/veto/internal/core"
	"github.com/melih-ucgun/veto/internal/inventory"
	"github.com/melih-ucgun/veto/internal/system"
	"github.com/melih-ucgun/veto/internal/transport"
	atomic "github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// fleetCmd represents the fleet command
var fleetCmd = &cobra.Command{
	Use:   "fleet",
	Short: "Manage a fleet of servers",
	Long:  `Inventory based operations for multiple hosts.`,
}

// factsCmd represents the facts command
var factsCmd = &cobra.Command{
	Use:   "facts",
	Short: "Gather system facts from all hosts",
	Run: func(cmd *cobra.Command, args []string) {
		// 1. Load Inventory
		invFile, _ := cmd.Flags().GetString("inventory")
		if invFile == "" {
			invFile = "inventory.yaml"
		}

		inv, err := inventory.LoadInventory(invFile)
		if err != nil {
			atomic.Error.Printf("Failed to load inventory: %v\n", err)
			return
		}

		// 2. Gather Facts concurrently
		type HostFact struct {
			HostName string
			OS       string
			Kernel   string
			CPU      string
			RAM      string
			Status   string
			Error    error
		}

		results := make(chan HostFact, len(inv.Hosts))
		var wg sync.WaitGroup

		spinner, _ := atomic.DefaultSpinner.Start(fmt.Sprintf("Gathering facts from %d hosts...", len(inv.Hosts)))

		for _, host := range inv.Hosts {
			wg.Add(1)
			go func(h inventory.Host) {
				defer wg.Done()

				// Map inventory.Host to config.Host for Transport
				// TODO: Load vars for Become info if needed
				cfgHost := config.Host{
					Name:         h.Name,
					Address:      h.Address,
					User:         h.User,
					Port:         h.Port,
					SSHKeyPath:   h.KeyPath,
					BecomeMethod: "sudo", // Default assumption for facts gathering
				}

				// Create Transport
				// TODO: Better context management with timeouts
				ctx := core.NewSystemContext(false, nil) // Base context

				// Initialize Transport
				var tr core.Transport
				var err error
				if h.Connection == "ssh" {
					tr, err = transport.NewSSHTransport(ctx.Context, cfgHost)
				} else {
					tr = transport.NewLocalTransport()
				}

				if err != nil {
					results <- HostFact{HostName: h.Name, Status: "OFFLINE", Error: err}
					return
				}
				defer tr.Close()

				// Update Context
				ctx.Transport = tr
				// CRITICAL: Set FS from Transport!
				ctx.FS = tr.GetFileSystem()

				// Run Detection
				// Panic safety?
				defer func() {
					if r := recover(); r != nil {
						results <- HostFact{HostName: h.Name, Status: "PANIC", Error: fmt.Errorf("%v", r)}
					}
				}()

				system.Detect(ctx)

				results <- HostFact{
					HostName: h.Name,
					OS:       fmt.Sprintf("%s %s", ctx.Distro, ctx.Version),
					Kernel:   ctx.Kernel,
					CPU:      ctx.Hardware.CPUModel,
					RAM:      ctx.Hardware.RAMTotal,
					Status:   "ONLINE",
				}
			}(host)
		}

		wg.Wait()
		close(results)
		spinner.Success("Facts gathered")

		// 3. Display Table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "HOST\tSTATUS\tOS\tKERNEL\tCPU\tRAM")
		fmt.Fprintln(w, "----\t------\t--\t------\t---\t---")

		for res := range results {
			statusIcon := "✅"
			if res.Status != "ONLINE" {
				statusIcon = "❌"
			}

			// Truncate CPU for display
			cpuDisplay := res.CPU
			if len(cpuDisplay) > 30 {
				cpuDisplay = cpuDisplay[:27] + "..."
			}

			fmt.Fprintf(w, "%s\t%s %s\t%s\t%s\t%s\t%s\n",
				res.HostName,
				statusIcon, res.Status,
				res.OS,
				res.Kernel,
				cpuDisplay,
				res.RAM,
			)
		}
		w.Flush()
	},
}

func init() {
	rootCmd.AddCommand(fleetCmd)
	fleetCmd.AddCommand(factsCmd)
	fleetCmd.PersistentFlags().StringP("inventory", "i", "inventory.yaml", "Path to inventory file")
}
