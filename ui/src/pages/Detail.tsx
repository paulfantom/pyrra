import { Link, RouteComponentProps, useHistory, useLocation } from 'react-router-dom'
import React, { useEffect, useMemo, useState } from 'react'
import { Button, ButtonGroup, Col, Container, Row, Spinner } from 'react-bootstrap'
import {
  Configuration,
  Objective,
  ObjectivesApi,
  ObjectiveStatus as APIObjectiveStatus,
  ObjectiveStatusAvailability,
  ObjectiveStatusBudget
} from '../client'
import { formatDuration, parseDuration, PUBLIC_API } from '../App'
import AlertsTable from '../components/AlertsTable'
import ErrorBudgetGraph from '../components/graphs/ErrorBudgetGraph'
import RequestsGraph from '../components/graphs/RequestsGraph'
import ErrorsGraph from '../components/graphs/ErrorsGraph'
import Navbar from '../components/Navbar'

interface DetailRouteParams {
  name: string
  namespace: string
}

const Detail = (params: RouteComponentProps<DetailRouteParams>) => {
    const { namespace, name } = params.match.params;

    const history = useHistory()
    const query = new URLSearchParams(useLocation().search)

    const api = useMemo(() => {
      return new ObjectivesApi(new Configuration({ basePath: `${PUBLIC_API}api/v1` }))
    }, [])

    const timeRangeQuery = query.get('timerange')
    const timeRangeParsed = timeRangeQuery != null ? parseDuration(timeRangeQuery) : null
    const timeRange: number = timeRangeParsed != null ? timeRangeParsed : 3600 * 1000

    const [objective, setObjective] = useState<Objective | null>(null);
    const [objectiveError, setObjectiveError] = useState<string>('');

    enum StatusState {
      Unknown,
      Error,
      NoData,
      Success,
    }

    const [statusState, setStatusState] = useState<StatusState>(StatusState.Unknown);
    const [availability, setAvailability] = useState<ObjectiveStatusAvailability | null>(null);
    const [errorBudget, setErrorBudget] = useState<ObjectiveStatusBudget | null>(null);

    useEffect(() => {
      const controller = new AbortController()

      document.title = `${name} - Pyrra`

      api.getObjective({ namespace, name })
        .then((o: Objective) => setObjective(o))
        .catch((resp) => {
          if (resp.status !== undefined) {
            resp.text().then((err: string) => (setObjectiveError(err)))
          } else {
            setObjectiveError(resp.message)
          }
        })

      api.getObjectiveStatus({ namespace, name })
        .then((s: APIObjectiveStatus) => {
          setAvailability(s.availability)
          setErrorBudget(s.budget)
          setStatusState(StatusState.Success)
        })
        .catch((resp) => {
          if (resp.status === 404) {
            setStatusState(StatusState.NoData)
          } else {
            setStatusState(StatusState.Error)
          }
        })

      return () => {
        // cancel any pending requests.
        controller.abort()
      }
    }, [api, namespace, name, timeRange])

    if (objectiveError !== '') {
      return (
        <>
          <Navbar/>
          <Container>
            <div style={{ margin: '50px 0' }}>
              <h3>{objectiveError}</h3>
              <br/>
              <Link to="/" className="btn btn-light">
                Go Back
              </Link>
            </div>
          </Container>
        </>
      )
    }

    if (objective == null) {
      return (
        <div style={{ marginTop: '50px', textAlign: 'center' }}>
          <Spinner animation="border" role="status">
            <span className="sr-only">Loading...</span>
          </Spinner>
        </div>
      )
    }

    const timeRanges = [
      28 * 24 * 3600 * 1000, // 4w
      7 * 24 * 3600 * 1000, // 1w
      24 * 3600 * 1000, // 1d
      12 * 3600 * 1000, // 12h
      3600 * 1000 // 1h
    ]

    const handleTimeRangeClick = (t: number) => () => {
      history.push(`/objectives/${namespace}/${name}?timerange=${formatDuration(t)}`)
    }

    const renderAvailability = () => {
      const headline = (<h6>Availability</h6>)
      switch (statusState) {
        case StatusState.Unknown:
          return (
            <div>
              {headline}
              <Spinner animation={'border'} style={{
                width: 50,
                height: 50,
                padding: 0,
                borderRadius: 50,
                borderWidth: 2,
                opacity: 0.25
              }}/>
            </div>
          )
        case StatusState.Error:
          return (
            <div>
              {headline}
              <h2 className="error">Error</h2>
            </div>
          )
        case StatusState.NoData:
          return (
            <div>
              {headline}
              <h2>No data</h2>
            </div>
          )
        case StatusState.Success:
          if (availability === null) {
            return <></>
          }
          return (
            <div className={availability.percentage > objective.target ? 'good' : 'bad'}>
              {headline}
              <h2>{(100 * availability.percentage).toFixed(3)}%</h2>
            </div>
          )
      }
    }

    const renderErrorBudget = () => {
      const headline = (<h6>Error Budget</h6>)
      switch (statusState) {
        case StatusState.Unknown:
          return (
            <div>
              {headline}
              <Spinner animation={'border'} style={{
                width: 50,
                height: 50,
                padding: 0,
                borderRadius: 50,
                borderWidth: 2,
                opacity: 0.25
              }}/>
            </div>
          )
        case StatusState.Error:
          return (
            <div>
              {headline}
              <h2 className="error">Error</h2>
            </div>
          )
        case StatusState.NoData:
          return (
            <div>
              {headline}
              <h2>No data</h2>
            </div>
          )
        case StatusState.Success:
          if (errorBudget === null) {
            return <></>
          }
          return (
            <div className={errorBudget.remaining > 0 ? 'good' : 'bad'}>
              {headline}
              <h2>{(100 * errorBudget.remaining).toFixed(3)}%</h2>
            </div>
          )
      }
    }

    return (
      <>
        <Navbar>
          <div>
            <Link to="/">Objectives</Link> &gt; <span>{objective.name}</span>
          </div>
        </Navbar>

        <div className="content detail">
          <Container>
            <Row>
              <Col xs={12}>
                <h3>{objective.name}</h3>
              </Col>
              {objective.description !== undefined && objective.description !== '' ? (
                  <Col xs={12} md={6}>
                    <p>{objective.description}</p>
                  </Col>
                )
                : (<></>)}
            </Row>
            <Row>
              <div className="metrics">
                <div>
                  <h6>Objective in <strong>{formatDuration(objective.window)}</strong></h6>
                  <h2>{(100 * objective.target).toFixed(3)}%</h2>
                </div>

                {renderAvailability()}

                {renderErrorBudget()}

              </div>
            </Row>
            <Row>
              <Col className="text-center timerange">
                <div className="inner">
                  <ButtonGroup aria-label="Time Range">
                    {timeRanges.map((t: number, i: number) => (
                      <Button
                        key={i}
                        variant="light"
                        onClick={handleTimeRangeClick(t)}
                        active={timeRange === t}
                      >{formatDuration(t)}</Button>
                    ))}
                  </ButtonGroup>
                </div>
              </Col>
            </Row>
            <Row style={{ marginBottom: 0 }}>
              <Col>
                <ErrorBudgetGraph
                  api={api}
                  namespace={namespace}
                  name={name}
                  timeRange={timeRange}
                />
              </Col>
            </Row>
            <Row>
              <Col style={{ textAlign: 'right' }}>
                {availability != null ? (
                  <>
                    <small>Errors: {Math.floor(availability.errors).toLocaleString()}</small>&nbsp;
                    <small>Total: {Math.floor(availability.total).toLocaleString()}</small>&nbsp;
                  </>
                ) : (
                  <></>
                )}
              </Col>
            </Row>
            <Row>
              <Col xs={12} sm={6}>
                <RequestsGraph
                  api={api}
                  namespace={namespace}
                  name={name}
                  timeRange={timeRange}
                />
              </Col>
              <Col xs={12} sm={6}>
                <ErrorsGraph
                  api={api}
                  namespace={namespace}
                  name={name}
                  timeRange={timeRange}
                />
              </Col>
            </Row>
            <Row>
              <Col>
                <h4>Multi Burn Rate Alerts</h4>
                <AlertsTable objective={objective}/>
              </Col>
            </Row>
            <Row>
              <Col>
                <h4>Config</h4>
                <pre style={{ padding: 20, borderRadius: 4 }}>
                <code>{objective.config}</code>
              </pre>
              </Col>
            </Row>
          </Container>
        </div>
      </>
    );
  }
;

export default Detail
