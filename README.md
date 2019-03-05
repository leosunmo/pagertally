# Pagertally
Pagertally queries the PagerDuty API to retrieve the rendered on-call schedules for the specified time period and schedule ID you provide and provides a breakdown of the time spent on-call during various times of the week. For example, a breakdown on how much time you spent on-call during a weekend period or a stat holiday. This would be used for compensation for engineers on-call.

### Statutory Holidays
Statutory holidays are added using an iCal url. This is simply because that's how the official list of statutory holidays are provided in New Zealand by the Ministry of Business, Innovation & Employment. Feel free to submit a PR to use local files or provide the dates in the config file.

You can whitelist which Stat days you want to honor in the configuration file.

### Config
The config is pretty straight forward.

 The `holidays` list is the list of statutory holidays you want to honor.

Business hours are configured in `business_hours`. This determines when on-call counts as "after hours" or when a weekend starts on Fridays.

The statutory holidays iCal is provided in `ical_url`.

`company_days` are arbitrary days your company decides is a holiday. The reason it's a separate type is because you might want to treat them differently from stat days.

The reason you can specify a `timezone` value is because some iCal files (like the NZ public holidays one) do **not** specify a timezone for the events, instead the calendar program has to make the decision of whether to convert to local time or not. By specifying a timezone you will forcibly add the timezone offset to the calendar events. 


```yaml
holidays:
  - "Christmas Day"
  - "Boxing Day"
  - "New Year's Day"
  - "Day after New Year's Day"
  - "Auckland Anniversary Day"
  - "Waitangi Day"
  - "Queen's Birthday"
  - "Labour Day"
company_days:
  - "24/12/2018"
  - "27/12/2018"
  - "28/12/2018"
  - "31/12/2018"
business_hours:
  start: "08:00"
  end: "17:30"
ical_url: "http://apps.employment.govt.nz/ical/public-holidays-all.ics"
timezone: "Pacific/Auckland"
```

### Installation
```
go get -u -v github.com/leosunmo/pagertally
```
Or
```
git clone git@github.com:leosunmo/pagertally.git
cd pagertally
make build
```


### Usage
It can either print out the results to the terminal or create a CSV file.
```
Usage of ./pagertally:
  -c, --config string                  (Optional) Provide config file path. Looks for "config.yaml" by default
      --csvdir string                  (Optional) Print as CSVs to this directory
      --google-safile string           (Optional) Google Service Account token JSON file
      --gsheetid string                (Optional) Print to Google Sheet ID provided
  -h, --help                           Print usage
  -m, --month string                   (Optional) Provide the month and year you want to process. Format: March 2018. Default: previous month
  -t, --pagerduty-token SecretString   PagerDuty API token (default [REDACTED])
  -s, --schedules strings              Comma separated list of PagerDuty schedule IDs

./pagerduty-shifts --pagerduty-token="pd-secret-token" --schedules SCHED1,SCHED2,SCHED3 --config conf.yaml [--month june] [--csvdir results.csv] | [--gsheetid GSheetID  --google-safile service-account.json]
```

### TODO
- [ ] Create a slack bot that you can interact with rather than using the command line or Cron.
- [ ] Probably look in to using https://github.com/senseyeio/spaniel for timespans
- [ ] Simple Kubernetes deployment?
- [ ] Add Google service account how-to
