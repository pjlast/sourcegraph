import React, { useMemo, useState } from 'react'

import { mdiChevronDoubleUp, mdiChevronDoubleDown } from '@mdi/js'
import classNames from 'classnames'
import * as H from 'history'

import { ContributableMenu } from '@sourcegraph/client-api'
import { SearchPatternTypeProps, CaseSensitivityProps } from '@sourcegraph/search'
import { ActionItem } from '@sourcegraph/shared/src/actions/ActionItem'
import { ActionsContainer } from '@sourcegraph/shared/src/actions/ActionsContainer'
import { ExtensionsControllerProps } from '@sourcegraph/shared/src/extensions/controller'
import { PlatformContextProps } from '@sourcegraph/shared/src/platform/context'
import { FilterKind, findFilter } from '@sourcegraph/shared/src/search/query/query'
import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { Button, Icon, Link } from '@sourcegraph/wildcard'

import { AuthenticatedUser } from '../../auth'
import { CloudCtaBanner } from '../../components/CloudCtaBanner'

import {
    getCodeMonitoringCreateAction,
    getInsightsCreateAction,
    getSearchContextCreateAction,
    getBatchChangeCreateAction,
    CreateAction,
} from './createActions'
import { SearchActionsMenu } from './SearchActionsMenu'

import styles from './SearchResultsInfoBar.module.scss'

export interface SearchResultsInfoBarProps
    extends ExtensionsControllerProps<'executeCommand' | 'extHostAPI'>,
        PlatformContextProps<'settings' | 'sourcegraphURL'>,
        TelemetryProps,
        SearchPatternTypeProps,
        Pick<CaseSensitivityProps, 'caseSensitive'> {
    history: H.History
    /** The currently authenticated user or null */
    authenticatedUser: Pick<AuthenticatedUser, 'id'> | null

    /**
     * Whether the code insights feature flag is enabled.
     */
    enableCodeInsights?: boolean
    enableCodeMonitoring: boolean

    /** The search query and if any results were found */
    query?: string
    resultsFound: boolean

    batchChangesEnabled?: boolean
    /** Whether running batch changes server-side is enabled */
    batchChangesExecutionEnabled?: boolean

    // Expand all feature
    allExpanded: boolean
    onExpandAllResultsToggle: () => void

    // Saved queries
    onSaveQueryClick: () => void

    location: H.Location

    className?: string

    stats: JSX.Element

    onShowMobileFiltersChanged?: (show: boolean) => void

    sidebarCollapsed: boolean
    setSidebarCollapsed: (collapsed: boolean) => void

    isSourcegraphDotCom: boolean
}

/**
 * The info bar shown over the search results list that displays metadata
 * and a few actions like expand all and save query
 */
export const SearchResultsInfoBar: React.FunctionComponent<
    React.PropsWithChildren<SearchResultsInfoBarProps>
> = props => {
    const globalTypeFilter = useMemo(
        () => (props.query ? findFilter(props.query, 'type', FilterKind.Global)?.value?.value : undefined),
        [props.query]
    )

    const canCreateMonitorFromQuery = useMemo(() => {
        if (globalTypeFilter) {
            return false
        }
        return globalTypeFilter === 'diff' || globalTypeFilter === 'commit'
    }, [globalTypeFilter])

    const canCreateBatchChangeFromQuery = useMemo(() => {
        if (!globalTypeFilter) {
            return true
        }
        return globalTypeFilter !== 'diff' && globalTypeFilter !== 'commit'
    }, [globalTypeFilter])

    // When adding a new create action check and update the $collapse-breakpoint in CreateActions.module.scss.
    // The collapse breakpoint indicates at which window size we hide the buttons and show the collapsed menu instead.
    const createActions = useMemo(
        () =>
            [
                getBatchChangeCreateAction(
                    props.query,
                    props.patternType,
                    Boolean(
                        props.batchChangesEnabled &&
                            props.batchChangesExecutionEnabled &&
                            props.authenticatedUser &&
                            canCreateBatchChangeFromQuery
                    )
                ),
                getSearchContextCreateAction(props.query, props.authenticatedUser),
                getInsightsCreateAction(
                    props.query,
                    props.patternType,
                    props.authenticatedUser,
                    props.enableCodeInsights
                ),
            ].filter((button): button is CreateAction => button !== null),
        [
            props.authenticatedUser,
            props.enableCodeInsights,
            props.patternType,
            props.query,
            props.batchChangesEnabled,
            props.batchChangesExecutionEnabled,
            canCreateBatchChangeFromQuery,
        ]
    )

    // The create code monitor action is separated from the rest of the actions, because we use the
    // <ExperimentalActionButton /> component instead of a regular (button) link, and it has a tour attached.
    const createCodeMonitorAction = useMemo(
        () => getCodeMonitoringCreateAction(props.query, props.patternType, props.enableCodeMonitoring),
        [props.enableCodeMonitoring, props.patternType, props.query]
    )

    const extraContext = useMemo(
        () => ({
            searchQuery: props.query || null,
            patternType: props.patternType,
            caseSensitive: props.caseSensitive,
        }),
        [props.query, props.patternType, props.caseSensitive]
    )

    // Show/hide mobile filters menu
    const [showMobileFilters, setShowMobileFilters] = useState(false)
    const onShowMobileFiltersClicked = (): void => {
        const newShowFilters = !showMobileFilters
        setShowMobileFilters(newShowFilters)
        props.onShowMobileFiltersChanged?.(newShowFilters)
    }

    const { extensionsController } = props

    return (
        <aside
            role="region"
            aria-label="Search results information"
            className={classNames(props.className, styles.searchResultsInfoBar)}
            data-testid="results-info-bar"
        >
            <div className={styles.row}>
                {props.stats}

                {props.isSourcegraphDotCom && (
                    <CloudCtaBanner className="mb-5" variant="outlined">
                        To search across your private repositories,{' '}
                        <Link
                            to="https://signup.sourcegraph.com"
                            target="_blank"
                            rel="noopener noreferrer"
                            onClick={() => props.telemetryService.log('ClickedOnCloudCTA')}
                        >
                            try Sourcegraph Cloud
                        </Link>
                        .
                    </CloudCtaBanner>
                )}

                <div className={styles.expander} />

                <ul className="nav align-items-center">
                    {extensionsController !== null && window.context.enableLegacyExtensions ? (
                        <ActionsContainer
                            {...props}
                            extensionsController={extensionsController}
                            extraContext={extraContext}
                            menu={ContributableMenu.SearchResultsToolbar}
                        >
                            {actionItems => (
                                <>
                                    {actionItems.map(actionItem => (
                                        <ActionItem
                                            {...props}
                                            {...actionItem}
                                            extensionsController={extensionsController}
                                            key={actionItem.action.id}
                                            showLoadingSpinnerDuringExecution={false}
                                            className="mr-2 text-decoration-none"
                                            actionItemStyleProps={{
                                                actionItemVariant: 'secondary',
                                                actionItemSize: 'sm',
                                                actionItemOutline: true,
                                            }}
                                        />
                                    ))}
                                </>
                            )}
                        </ActionsContainer>
                    ) : null}

                    <li className={styles.divider} aria-hidden="true" />

                    <SearchActionsMenu
                        query={props.query}
                        patternType={props.patternType}
                        sourcegraphURL={props.platformContext.sourcegraphURL}
                        authenticatedUser={props.authenticatedUser}
                        createActions={createActions}
                        createCodeMonitorAction={createCodeMonitorAction}
                        canCreateMonitor={canCreateMonitorFromQuery}
                        resultsFound={props.resultsFound}
                        allExpanded={props.allExpanded}
                        onExpandAllResultsToggle={props.onExpandAllResultsToggle}
                        onSaveQueryClick={props.onSaveQueryClick}
                    />
                </ul>

                <Button
                    className={classNames(
                        'd-flex align-items-center d-lg-none',
                        styles.filtersButton,
                        showMobileFilters && 'active'
                    )}
                    aria-pressed={showMobileFilters}
                    onClick={onShowMobileFiltersClicked}
                    outline={true}
                    variant="secondary"
                    size="sm"
                    aria-label={`${showMobileFilters ? 'Hide' : 'Show'} filters`}
                >
                    Filters
                    <Icon
                        aria-hidden={true}
                        className="ml-2"
                        svgPath={showMobileFilters ? mdiChevronDoubleUp : mdiChevronDoubleDown}
                    />
                </Button>

                {props.sidebarCollapsed && (
                    <Button
                        className={classNames('align-items-center d-none d-lg-flex', styles.filtersButton)}
                        onClick={() => props.setSidebarCollapsed(false)}
                        outline={true}
                        variant="secondary"
                        size="sm"
                        aria-label="Show filters sidebar"
                    >
                        Filters
                        <Icon aria-hidden={true} className="ml-2" svgPath={mdiChevronDoubleDown} />
                    </Button>
                )}
            </div>
        </aside>
    )
}
