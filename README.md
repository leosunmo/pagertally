# Pagerduty-schedule
This little program queries the Pagerduty API to retrieve the on-call shifts in the previous month from any schedule ID you provide it.

### Hours
It has a concept of "shift hours" where it basically breaks down all user shifts in to hours. This means that it will sometimes round up and sometimes round down when calculating a shift.
If you've done a 7 Hours and 31 minute shift, it'll count it as 8 hours. If you've done a 7 Hours and 29 minute shift, it'll round it down to 7 hours.

All hours are tagged as one of four types of hours.
* Business hours
* After hours
* Weekend hours
* Statutory Holiday hours

### Statutory Holidays
Statutory holidays are added using an iCal url. This is simply because that's how the official list of statutory holidays are provided in New Zealand by the Ministry of Business, Innovation & Employment. Feel free to submit a PR to use local files or provide the dates in the config file.

You can whitelist which Stat days you want to honor in the configuration file.

### Config
The config is pretty straight forward. The `holidays` list is the list of statutory holidays you want to honor. 

Business hours are configured in `business_hours`. This determines when on-call counts as "after hours". 

The iCal is provided in `ical_url`. 

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
business_hours:
  start: "09:00"
  end: "18:00"
ical_url: "http://apps.employment.govt.nz/ical/public-holidays-all.ics"
timezone: "Pacific/Auckland"
```

### Installation
```
go get -u -v github.com/leosunmo/pagerduty-schedule
```
Or
```
git clone git@github.com:leosunmo/pagerduty-schedule.git
cd pagerduty-schedule
go build
```


### Usage
It can either print out the results to the terminal or create a CSV file.
```
Usage of ./pagerduty-schedule:
	-conf string
		Provide config file path
	-outfile string
		(Optional) Print as CSV to this file
	-schedule string
		Provide PagerDuty schedule ID
	-token string
		Provide PagerDuty API token

./pagerduty-schedule -token="my-secret-token" -schedule MYSCHEDULEID -conf conf.yaml [-outfile results.csv]
```
