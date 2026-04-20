package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/local/picobot/internal/agent/memory"
	"github.com/local/picobot/internal/config"
	"github.com/local/picobot/internal/providers"
)

func newMemoryCmd() *cobra.Command {
	// memory subcommands: read, append, write, recent
	memoryCmd := &cobra.Command{
		Use:   "memory",
		Short: "Inspect or modify workspace memory files",
	}

	readCmd := &cobra.Command{
		Use:   "read [today|long]",
		Short: "Read memory (today or long-term)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			target := args[0]
			cfg, _ := config.LoadConfig()
			ws := cfg.Agents.Defaults.Workspace
			if ws == "" {
				ws = "~/.picobot/workspace"
			}
			home, _ := os.UserHomeDir()
			if strings.HasPrefix(ws, "~/") {
				ws = filepath.Join(home, ws[2:])
			}
			mem := memory.NewMemoryStoreWithWorkspace(ws, 100)
			switch target {
			case "today":
				out, _ := mem.ReadToday()
				fmt.Fprintln(cmd.OutOrStdout(), out)
			case "long":
				out, _ := mem.ReadLongTerm()
				fmt.Fprintln(cmd.OutOrStdout(), out)
			default:
				fmt.Fprintln(cmd.ErrOrStderr(), "unknown target: "+target)
			}
		},
	}

	appendCmd := &cobra.Command{
		Use:   "append [today|long] -c <content>",
		Short: "Append content to today's note or long-term memory",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			target := args[0]
			content, _ := cmd.Flags().GetString("content")
			if content == "" {
				fmt.Fprintln(cmd.ErrOrStderr(), "-c content required")
				return
			}
			cfg, _ := config.LoadConfig()
			ws := cfg.Agents.Defaults.Workspace
			if ws == "" {
				ws = "~/.picobot/workspace"
			}
			home, _ := os.UserHomeDir()
			if strings.HasPrefix(ws, "~/") {
				ws = filepath.Join(home, ws[2:])
			}
			mem := memory.NewMemoryStoreWithWorkspace(ws, 100)
			switch target {
			case "today":
				if err := mem.AppendToday(content); err != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), "append failed:", err)
					return
				}
				fmt.Fprintln(cmd.OutOrStdout(), "appended to today")
			case "long":
				lt, err := mem.ReadLongTerm()
				if err != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), "append long failed:", err)
					return
				}
				if err := mem.WriteLongTerm(lt + "\n" + content); err != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), "append long failed:", err)
					return
				}
				fmt.Fprintln(cmd.OutOrStdout(), "appended to long-term memory")
			default:
				fmt.Fprintln(cmd.ErrOrStderr(), "unknown target:", target)
			}
		},
	}
	appendCmd.Flags().StringP("content", "c", "", "Content to append")

	writeCmd := &cobra.Command{
		Use:   "write long -c <content>",
		Short: "Write (overwrite) long-term MEMORY.md",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if args[0] != "long" {
				fmt.Fprintln(os.Stderr, "write currently only supports 'long'")
				return
			}
			content, _ := cmd.Flags().GetString("content")
			if content == "" {
				fmt.Fprintln(cmd.ErrOrStderr(), "-c content required")
				return
			}
			cfg, _ := config.LoadConfig()
			ws := cfg.Agents.Defaults.Workspace
			if ws == "" {
				ws = "~/.picobot/workspace"
			}
			home, _ := os.UserHomeDir()
			if strings.HasPrefix(ws, "~/") {
				ws = filepath.Join(home, ws[2:])
			}
			mem := memory.NewMemoryStoreWithWorkspace(ws, 100)
			if err := mem.WriteLongTerm(content); err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), "write failed:", err)
				return
			}
			fmt.Fprintln(cmd.OutOrStdout(), "wrote long-term memory")
		},
	}
	writeCmd.Flags().StringP("content", "c", "", "Content to write")

	recentCmd := &cobra.Command{
		Use:   "recent -days N",
		Short: "Show recent N days' notes",
		Run: func(cmd *cobra.Command, args []string) {
			days, _ := cmd.Flags().GetInt("days")
			cfg, _ := config.LoadConfig()
			ws := cfg.Agents.Defaults.Workspace
			if ws == "" {
				ws = "~/.picobot/workspace"
			}
			home, _ := os.UserHomeDir()
			if strings.HasPrefix(ws, "~/") {
				ws = filepath.Join(home, ws[2:])
			}
			mem := memory.NewMemoryStoreWithWorkspace(ws, 100)
			out, _ := mem.GetRecentMemories(days)
			fmt.Fprintln(cmd.OutOrStdout(), out)
		},
	}
	recentCmd.Flags().IntP("days", "d", 1, "Number of days to include")

	memoryCmd.AddCommand(readCmd)
	memoryCmd.AddCommand(appendCmd)
	memoryCmd.AddCommand(writeCmd)
	memoryCmd.AddCommand(recentCmd)

	// rank subcommand: rank recent memories by relevance to a query
	rankCmd := &cobra.Command{
		Use:   "rank -q <query>",
		Short: "Rank recent memories relative to a query",
		Run: func(cmd *cobra.Command, args []string) {
			q, _ := cmd.Flags().GetString("query")
			if q == "" {
				fmt.Fprintln(cmd.ErrOrStderr(), "-q query required")
				return
			}
			top, _ := cmd.Flags().GetInt("top")
			verbose, _ := cmd.Flags().GetBool("verbose")
			cfg, _ := config.LoadConfig()
			ws := cfg.Agents.Defaults.Workspace
			if ws == "" {
				ws = "~/.picobot/workspace"
			}
			home, _ := os.UserHomeDir()
			if strings.HasPrefix(ws, "~/") {
				ws = filepath.Join(home, ws[2:])
			}
			mem := memory.NewMemoryStoreWithWorkspace(ws, 100)
			// Build memory items from today's file (split into lines) and long-term memory
			items := make([]memory.MemoryItem, 0)
			if td, err := mem.ReadToday(); err == nil && td != "" {
				for _, line := range strings.Split(td, "\n") {
					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}
					// strip leading timestamp [2026-02-07...] if present
					if idx := strings.Index(line, "] "); idx != -1 && strings.HasPrefix(line, "[") {
						line = strings.TrimSpace(line[idx+2:])
					}
					items = append(items, memory.MemoryItem{Kind: "today", Text: line})
				}
			}
			if lt, err := mem.ReadLongTerm(); err == nil && lt != "" {
				for _, line := range strings.Split(lt, "\n") {
					line = strings.TrimSpace(line)
					if line == "" {
						continue
					}
					items = append(items, memory.MemoryItem{Kind: "long", Text: line})
				}
			}
			provider := providers.NewProviderFromConfig(cfg)
			var logger *log.Logger
			if verbose {
				logger = log.New(cmd.OutOrStdout(), "ranker: ", 0)
			}
			ranker := memory.NewLLMRankerWithLogger(provider, provider.GetDefaultModel(), logger)
			res := ranker.Rank(q, items, top)
			for i, m := range res {
				fmt.Fprintf(cmd.OutOrStdout(), "%d: %s (%s)\n", i+1, m.Text, m.Kind)
			}
		},
	}
	rankCmd.Flags().StringP("query", "q", "", "Query to rank memories against")
	rankCmd.Flags().IntP("top", "k", 5, "Number of top memories to show")
	rankCmd.Flags().BoolP("verbose", "v", false, "Enable verbose diagnostic logging (to stdout)")
	memoryCmd.AddCommand(rankCmd)

	return memoryCmd
}
