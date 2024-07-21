package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/aws/aws-lambda-go/lambda"
)

func handleRequest(ctx context.Context) (string, error) {
	targetURL := os.Getenv("TARGET_URL")
	if targetURL == "" {
		return "", fmt.Errorf("TARGET_URL environment variable is not set")
	}

	memberID := os.Getenv("SLACK_MEMBER_ID")
	if memberID == "" {
		log.Println("SLACK_MEMBER_ID is not set, notification will be sent without mention")
	}

	inStock, err := checkStock(targetURL)
	if err != nil {
		return "", err
	}

	if inStock {
		message := fmt.Sprintf("商品在庫あります！\n商品ページ: %s", targetURL)
		err = sendSlackNotification(message, targetURL, memberID)
		if err != nil {
			log.Printf("Failed to send Slack notification: %v", err)
			return "", err
		}
		return message, nil
	}

	return "商品は在庫切れです。通知は送信されませんでした。", nil
}

func checkStock(url string) (bool, error) {
	resp, err := http.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, fmt.Errorf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return false, err
	}

	button := doc.Find("button[name='add']")
	isDisabled := button.AttrOr("disabled", "") != ""

	soldOutText := button.Find("span").Text()
	isSoldOut := strings.Contains(strings.ToUpper(soldOutText), "SOLD OUT")

	inStock := !isDisabled && !isSoldOut

	return inStock, nil
}

func sendSlackNotification(message, targetURL, memberID string) error {
	webhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	if webhookURL == "" {
		return fmt.Errorf("SLACK_WEBHOOK_URL is not set")
	}

	mentionText := ""
	if memberID != "" {
		mentionText = fmt.Sprintf("<@%s> ", memberID)
	}

	payload := map[string]interface{}{
		"text": fmt.Sprintf("%s%s", mentionText, message),
		"blocks": []map[string]interface{}{
			{
				"type": "section",
				"text": map[string]string{
					"type": "mrkdwn",
					"text": fmt.Sprintf("%s*商品在庫あります！*", mentionText),
				},
			},
			{
				"type": "section",
				"text": map[string]string{
					"type": "mrkdwn",
					"text": fmt.Sprintf("商品ページ: <%s|こちらをクリック>", targetURL),
				},
			},
		},
	}

	slackBody, _ := json.Marshal(payload)
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(slackBody))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func main() {
	lambda.Start(handleRequest)
}
