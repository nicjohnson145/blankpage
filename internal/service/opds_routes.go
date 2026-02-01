package service

import "fmt"
import "net/url"

func RouteRoot() string {
	return "/opds"
}

func filterRouteTemplate() string {
	return RouteRoot() + "/filter/%v"
}

func RouteFilterPattern() string {
	return fmt.Sprintf(filterRouteTemplate(), "{kind}")
}

func routeFilterKindSeries() string {
	return "series"
}

func routeFilterKindRecentlyAdded() string {
	return "recent"
}

func routeFilterKindAllAlphabetical() string {
	return "all-alphabetical"
}

func routeFilterSeries() string {
	return fmt.Sprintf(filterRouteTemplate(), routeFilterKindSeries())
}

func routeFilterRecentlyAdded() string {
	return fmt.Sprintf(filterRouteTemplate(), routeFilterKindRecentlyAdded())
}

func routeFilterAllAlphabetical() string {
	return fmt.Sprintf(filterRouteTemplate(), routeFilterKindAllAlphabetical())
}

func RouteSingleSeriesPattern() string {
	return routeFilterSeries() + "/{series_name}"
}

func routeSingleSeries(name string) string {
	return routeFilterSeries() + "/" + url.PathEscape(name)
}

func downloadRouteTemplate() string {
	return RouteRoot() + "/download/%v/%v"
}

func routeDownloadEpub(id string) string {
	return fmt.Sprintf(downloadRouteTemplate(), id, "epub")
}

func RouteDownloadPattern() string {
	return fmt.Sprintf(downloadRouteTemplate(), "{id}", "{format}")
}
