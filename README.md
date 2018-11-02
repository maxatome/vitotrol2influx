# Viessmann™ Vitotrol™ device to influx DB

Typically used to inject Viessmann™ Vitotrol™ boiler data into Influx
database.

## Installation

go 1.11 is needed to correctly handle dependencies.

```sh
go build -v
```

will generate a `vitotrol2influx` executable.


## Usage

Create a YAML file based on
[`vitotrol2influx.yml`](vitotrol2influx.yml), including your
credentials and the attributes you want to store in InfluxDB.

Registered attributes can be found here:
https://github.com/maxatome/go-vitotrol/blob/master/attributes.go#L79
(field `Name`).

If you want an attribute that is not registered, use a name like
`NAME_0xNNNN` where `NAME` is the name of the attribute, and `NNNN` is
the hexadecimal representation of the attribute ID, for example:

```
FuelConsumption_0x108d
```

Note that you can provide the special attribute
`ComputedSetpointTemp`. This fake attribute is computed using several
others and corresponds to the setpoint temperature (as it appears that
this value is not available in Vitotrol™ served attributes).

Once your `vitotrol2influx.yml` is ready, you can launch:

```sh
vitotrol2influx -config vitotrol2influx.yml
```

That's all.


## Future

I will add a (very) simple REST API to serve cached values and so
avoid the huge amount of time needed to retrieve them from the
Vitotrol™ web service.
