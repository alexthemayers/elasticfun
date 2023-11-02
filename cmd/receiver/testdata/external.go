package testdata

func CallExternalApi(ctx context.Context, client *http.Client, url string) (TestData, error) {
	workResult := rand.Int()
	if workResult%5 == 1 {
		// fail
		return TestData{}, fmt.Errorf("failure to do work")
	}
	// succeed
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return TestData{}, fmt.Errorf("failure to make api call: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return TestData{}, fmt.Errorf("failure to make api call: %w", err)
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return TestData{}, fmt.Errorf("failure to make api call: %w", err)
	}
	var activityStruct Activity
	err = json.Unmarshal(bodyBytes, &activityStruct)
	if err != nil {
		return TestData{}, fmt.Errorf("failure to make api call: %w", err)
	}
	return TestData{Activity: activityStruct.Activity, RandomNumber: workResult}, nil
}

type Activity struct {
	Activity      string  `json:"activity"`
	Type          string  `json:"type"`
	Participants  int     `json:"participants"`
	Price         float64 `json:"price"`
	Link          string  `json:"link"`
	Key           string  `json:"key"`
	Accessibility float64 `json:"accessibility"`
}

type TestData struct {
	Activity     string `json:"activity"`
	RandomNumber int    `json:"random_number"`
}
