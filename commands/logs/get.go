package logs

import (
	"context"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/AaronLieb/goat/aws"
	"github.com/AaronLieb/goat/ioutil"
	"github.com/AaronLieb/goat/util"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	json "github.com/neilotoole/jsoncolor"
	"github.com/urfave/cli/v3"
)

const cacheFile = "logs_get"

func GetSubcommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "get",
		Usage: "Query CloudWatch logs",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "group",
				Aliases: []string{"g"},
				Usage:   "The pattern for the log group",
			},
			&cli.TimestampFlag{
				Name:    "start",
				Aliases: []string{"s"},
				Usage:   "The start time for the query",
				Config: cli.TimestampConfig{
					Timezone: time.UTC,
					Layouts:  []string{"2006-01-02 15:04:05"},
				},
			},
			&cli.TimestampFlag{
				Name:    "end",
				Aliases: []string{"e"},
				Usage:   "The end time for the query",
				Config: cli.TimestampConfig{
					Timezone: time.UTC,
					Layouts:  []string{"2006-01-02 15:04:05"},
				},
			},
			&cli.DurationFlag{
				Name:    "relative",
				Aliases: []string{"r"},
				Usage:   "The relative duration of the query",
				Value:   time.Hour,
			},
			&cli.StringFlag{
				Name:    "query",
				Aliases: []string{"q"},
				Usage:   "The query string",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "The output format",
				Value:   "json",
			},
			&cli.BoolFlag{
				Name:    "cached",
				Aliases: []string{"c"},
				Usage:   "Fetch the cached result",
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Aliases: []string{"v"},
				Usage:   "Additional logging and messages",
			},
			&cli.BoolFlag{
				Name:    "debug",
				Aliases: []string{"d"},
				Usage:   "debug messages",
			},
			&cli.BoolFlag{
				Name:  "no-color",
				Usage: "no colored output",
				Value: false,
			},
		},
		Action: getLogs,
	}
	defaultComplete := cli.DefaultCompleteWithFlags

	cmd.ShellComplete = func(ctx context.Context, cmd *cli.Command) {
		if autocomplete(ctx, cmd) {
			return
		}
		defaultComplete(ctx, cmd)
	}
	return cmd
}

func getLogs(ctx context.Context, cmd *cli.Command) error {

	logGroupName := cmd.String("group")
	queryStartTime := cmd.Timestamp("start")
	queryEndTime := cmd.Timestamp("end")
	queryRelativeTime := cmd.Duration("relative")
	queryString := cmd.String("query")
	outputFormat := cmd.String("output")
	cached := cmd.Bool("cached")
	verbose := cmd.Bool("verbose")
	debug := cmd.Bool("debug")
	noColor := cmd.Bool("no-color")

	if util.DisableDoubleDash(ctx, cmd) {
		return nil
	}

	// Load config, instantiate CW client
	cfg, err := aws.LoadConfig(ctx, debug)
	util.PanicIfErr(err)

	// TODO: add some auth error checking

	client := cloudwatchlogs.NewFromConfig(cfg)

	// If --cached flag is provided, reach last result from cache
	if cached {
		printCached()
		return nil
	}

	// Calculate relative start time if no absolute time was provided
	if queryStartTime.IsZero() {
		start := time.Now().UTC().Add(-queryRelativeTime)
		queryStartTime = start
	}

	// Default end time is now, if no absolute time was provided
	if queryEndTime.IsZero() {
		now := time.Now().UTC()
		queryEndTime = now
	}

	if verbose {
		fmt.Fprintf(cmd.ErrWriter, "Start: %s, End: %s\n", queryStartTime.Format(time.RFC3339), queryEndTime.Format(time.RFC3339))
		fmt.Fprintln(cmd.ErrWriter, queryString)
	}

	// TODO: Change logGroupName to optional, add fzf choosing of log groups
	if queryString == "" {
		return cli.Exit("--query is required unless --cached is specified", 1)
	}

	describeLogGroupsOutput, err := client.DescribeLogGroups(ctx, &cloudwatchlogs.DescribeLogGroupsInput{
		LogGroupNamePattern: &logGroupName,
	})
	util.PanicIfErr(err)

	// TODO: If a single log group is not found, we should use go-fzf
	// TODO: Instead of throwing an error, use go-fzf to select a log group
	if len(describeLogGroupsOutput.LogGroups) > 1 {
		return cli.Exit("More than one log group found for pattern", 1)
	}

	if len(describeLogGroupsOutput.LogGroups) == 0 {
		return cli.Exit("No log groups found for pattern", 1)
	}

	if verbose {
		fmt.Fprintf(cmd.ErrWriter, "logGroupName: %s\n", logGroupName)
	}

	// convert times to unix timestamps
	queryStartTimeUnix := queryStartTime.Unix()
	queryEndTimeUnix := queryEndTime.Unix()

	startQueryOutput, err := client.StartQuery(ctx, &cloudwatchlogs.StartQueryInput{
		LogGroupName: &logGroupName,
		StartTime:    &queryStartTimeUnix,
		EndTime:      &queryEndTimeUnix,
		QueryString:  &queryString,
	})
	util.PanicIfErr(err)

	// TODO: Split this into a separate function, or consider using aws sdk waiters
	var results [][]types.ResultField
	for i := 0; ; i++ {
		getQueryResultsOutput, err := client.GetQueryResults(ctx, &cloudwatchlogs.GetQueryResultsInput{
			QueryId: startQueryOutput.QueryId,
		})
		util.PanicIfErr(err)

		time.Sleep(time.Duration(backoff(3, i)) * time.Second)

		status := getQueryResultsOutput.Status
		if status == types.QueryStatusComplete {
			results = getQueryResultsOutput.Results
			break
		}

		fmt.Fprint(cmd.ErrWriter, "\r          \r")
		fmt.Fprint(cmd.ErrWriter, status)
	}
	fmt.Fprint(cmd.ErrWriter, "\r          \r")

	f, err := ioutil.OpenCache(cacheFile)
	defer f.Close()

	isColor := !noColor && json.IsColorTerminal(cmd.Writer)
	mw := io.MultiWriter(cmd.Writer, f)
	mwEnc := ioutil.GetEncoder(&mw, isColor)

	switch outputFormat {
	case "raw":
		mwEnc.Encode(results)
	case "fulljson":
		mwEnc.Encode(ioutil.Flatten(results))
	case "json":
		for _, log := range ioutil.Flatten(results) {
			mwEnc.Encode(log)
		}
	case "csv":
		ioutil.PrintCsv(&mw, results)
	}

	return nil
}

func backoff(minimum int, attempts int) int {
	return minimum + int(math.Log(1+float64(attempts)/math.Log(1.2)))
}

func autocomplete(ctx context.Context, cmd *cli.Command) bool {
	if cmd.NArg() < 2 {
		last := ""
		if cmd.NArg() > 0 {
			last = cmd.Args().Get(cmd.NArg() - 1)
		}
		cfg, err := aws.LoadConfig(ctx, false)
		if err != nil {
			return false
		}
		accessKeyId, err := aws.AccessKey(ctx, cfg)
		if err != nil {
			return false
		}
		cacheName := "log_groups_" + accessKeyId
		b, err := ioutil.ReadCache(cacheName)
		if err != nil {
			client := cloudwatchlogs.NewFromConfig(cfg)
			result, err := client.DescribeLogGroups(ctx, &cloudwatchlogs.DescribeLogGroupsInput{})
			if err != nil {
				panic(err)
			}
			logGroups := result.LogGroups
			for nextToken := result.NextToken; nextToken != nil; nextToken = result.NextToken {
				result, err = client.DescribeLogGroups(ctx, &cloudwatchlogs.DescribeLogGroupsInput{
					NextToken: nextToken,
				})
				if err != nil {
					panic(err)
				}
				logGroups = append(logGroups, result.LogGroups...)
			}
			if err != nil {
				return false
			}
			f, err := ioutil.OpenCache(cacheName)
			if err != nil {
				return false
			}
			for _, logGroup := range logGroups {
				if strings.Contains(*logGroup.LogGroupName, last) {
					fmt.Fprintln(cmd.Writer, *logGroup.LogGroupName)
				}
				fmt.Fprintln(f, *logGroup.LogGroupName)
			}
			return true
		}

		lines := strings.Split(string(b), "\n")
		for _, line := range lines {
			if strings.Contains(line, last) {
				fmt.Fprintln(cmd.Writer, line)
			}
		}
		return true
	}
	return false
}

func printCached() {
	// Might want to replace this with bufio chunked reading for memory efficiency
	data, err := ioutil.ReadCache(cacheFile)
	if err != nil {
		cli.Exit("Unable to read cached file.", 1)
	}

	fmt.Print(string(data))
}
