// Copyright 2020 The Terraformer Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package aws

import (
	"context"

	"github.com/GoogleCloudPlatform/terraformer/terraformutils"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchevents"
)

var cloudwatchAllowEmptyValues = []string{"tags."}

type CloudWatchGenerator struct {
	AWSService
}

func (g *CloudWatchGenerator) InitResources() error {
	config, e := g.generateConfig()
	if e != nil {
		return e
	}

	cloudwatchSvc := cloudwatch.NewFromConfig(config)
	err := g.createMetricAlarms(cloudwatchSvc)
	if err != nil {
		return err
	}
	err = g.createDashboards(cloudwatchSvc)
	if err != nil {
		return err
	}

	cloudwatcheventsSvc := cloudwatchevents.NewFromConfig(config)
	err = g.createRules(cloudwatcheventsSvc)
	if err != nil {
		return err
	}

	return nil
}

func (g *CloudWatchGenerator) createMetricAlarms(cloudwatchSvc *cloudwatch.Client) error {
	var nextToken *string
	for {
		output, err := cloudwatchSvc.DescribeAlarms(context.TODO(), &cloudwatch.DescribeAlarmsInput{
			NextToken: nextToken,
		})
		if err != nil {
			return err
		}
		for _, metricAlarm := range output.MetricAlarms {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				*metricAlarm.AlarmName,
				*metricAlarm.AlarmName,
				"aws_cloudwatch_metric_alarm",
				"aws",
				cloudwatchAllowEmptyValues))
		}
		nextToken = output.NextToken
		if nextToken == nil {
			break
		}
	}
	return nil
}

func (g *CloudWatchGenerator) createDashboards(cloudwatchSvc *cloudwatch.Client) error {
	var nextToken *string
	for {
		output, err := cloudwatchSvc.ListDashboards(context.TODO(), &cloudwatch.ListDashboardsInput{
			NextToken: nextToken,
		})
		if err != nil {
			return err
		}
		for _, dashboardEntry := range output.DashboardEntries {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				*dashboardEntry.DashboardName,
				*dashboardEntry.DashboardName,
				"aws_cloudwatch_dashboard",
				"aws",
				cloudwatchAllowEmptyValues))
		}
		nextToken = output.NextToken
		if nextToken == nil {
			break
		}
	}
	return nil
}

func (g *CloudWatchGenerator) createRules(cloudwatcheventsSvc *cloudwatchevents.Client) error {
	var listEventBusesNextToken *string
	for {
		eventBusesResponse, err := cloudwatcheventsSvc.ListEventBuses(context.TODO(), &cloudwatchevents.ListEventBusesInput{
			NextToken: listEventBusesNextToken,
		})
		if err != nil {
			return err
		}
		for _, eventBus := range eventBusesResponse.EventBuses {
			g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
				*eventBus.Name,
				*eventBus.Name,
				"aws_cloudwatch_event_bus",
				"aws",
				cloudwatchAllowEmptyValues,
			))
			var listRulesNextToken *string
			for {
				output, err := cloudwatcheventsSvc.ListRules(context.TODO(), &cloudwatchevents.ListRulesInput{
					EventBusName: eventBus.Name,
					NextToken:    listRulesNextToken,
				})
				if err != nil {
					return err
				}
				for _, rule := range output.Rules {
					ruleRef := *eventBus.Name + "/" + *rule.Name
					g.Resources = append(g.Resources, terraformutils.NewSimpleResource(
						ruleRef,
						ruleRef,
						"aws_cloudwatch_event_rule",
						"aws",
						cloudwatchAllowEmptyValues,
					))
					var listTargetsNextToken *string
					for {
						targetResponse, err := cloudwatcheventsSvc.ListTargetsByRule(context.TODO(), &cloudwatchevents.ListTargetsByRuleInput{
							Rule:         rule.Name,
							NextToken:    listTargetsNextToken,
							EventBusName: eventBus.Name,
						})
						if err != nil {
							return err
						}

						for _, target := range targetResponse.Targets {
							targetRef := *rule.Name + "/" + *target.Id
							g.Resources = append(g.Resources, terraformutils.NewResource(
								targetRef,
								targetRef,
								"aws_cloudwatch_event_target",
								"aws",
								map[string]string{
									"rule":           *rule.Name,
									"target_id":      *target.Id,
									"event_bus_name": *eventBus.Name,
								},
								cloudwatchAllowEmptyValues,
								map[string]interface{}{}))
						}
						listTargetsNextToken = targetResponse.NextToken
						if listTargetsNextToken == nil {
							break
						}
					}
				}
				listRulesNextToken = output.NextToken
				if listRulesNextToken == nil {
					break
				}
			}
		}
		listEventBusesNextToken = eventBusesResponse.NextToken
		if listEventBusesNextToken == nil {
			break
		}
	}

	return nil
}
