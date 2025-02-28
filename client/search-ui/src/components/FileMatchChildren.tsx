import React, { useCallback, useMemo, KeyboardEvent, MouseEvent } from 'react'

import classNames from 'classnames'
import { useHistory } from 'react-router'
import { Observable } from 'rxjs'
import { map } from 'rxjs/operators'

import { HoverMerged } from '@sourcegraph/client-api'
import { Hoverifier } from '@sourcegraph/codeintellify'
import {
    appendLineRangeQueryParameter,
    appendSubtreeQueryParameter,
    isErrorLike,
    toPositionOrRangeQueryParameter,
} from '@sourcegraph/common'
import { HighlightLineRange, HighlightResponseFormat } from '@sourcegraph/search'
import { ActionItemAction } from '@sourcegraph/shared/src/actions/ActionItem'
import { FetchFileParameters } from '@sourcegraph/shared/src/backend/file'
import { MatchGroup } from '@sourcegraph/shared/src/components/ranking/PerFileResultRanking'
import { Controller as ExtensionsController } from '@sourcegraph/shared/src/extensions/controller'
import { HoverContext } from '@sourcegraph/shared/src/hover/HoverOverlay.types'
import { ContentMatch, getFileMatchUrl } from '@sourcegraph/shared/src/search/stream'
import { SettingsCascadeProps } from '@sourcegraph/shared/src/settings/settings'
import { TelemetryProps } from '@sourcegraph/shared/src/telemetry/telemetryService'
import { codeCopiedEvent } from '@sourcegraph/shared/src/tracking/event-log-creators'
import { useCodeIntelViewerUpdates } from '@sourcegraph/shared/src/util/useCodeIntelViewerUpdates'
import { useFocusOnLoadedMore } from '@sourcegraph/wildcard'

import { CodeExcerpt } from './CodeExcerpt'
import { navigateToCodeExcerpt, navigateToFileOnMiddleMouseButtonClick } from './codeLinkNavigation'
import { LastSyncedIcon } from './LastSyncedIcon'

import styles from './FileMatchChildren.module.scss'

interface FileMatchProps extends SettingsCascadeProps, TelemetryProps {
    result: ContentMatch
    grouped: MatchGroup[]
    /* Clicking on a match opens the link in a new tab */
    openInNewTab?: boolean
    fetchHighlightedFileLineRanges: (parameters: FetchFileParameters, force?: boolean) => Observable<string[][]>
    extensionsController?: Pick<ExtensionsController, 'extHostAPI'> | null
    hoverifier?: Hoverifier<HoverContext, HoverMerged, ActionItemAction>
}

export const FileMatchChildren: React.FunctionComponent<React.PropsWithChildren<FileMatchProps>> = props => {
    /**
     * If LazyFileResultSyntaxHighlighting is enabled, we fetch plaintext
     * line ranges _alongside_ the typical highlighted line ranges.
     */
    const enableLazyFileResultSyntaxHighlighting =
        props.settingsCascade.final &&
        !isErrorLike(props.settingsCascade.final) &&
        props.settingsCascade.final.experimentalFeatures?.enableLazyFileResultSyntaxHighlighting

    const { result, grouped, fetchHighlightedFileLineRanges, telemetryService, extensionsController } = props

    const fetchFileRangeMatches = useCallback(
        (args: { format?: HighlightResponseFormat; ranges: HighlightLineRange[] }): Observable<string[][]> =>
            fetchHighlightedFileLineRanges(
                {
                    repoName: result.repository,
                    commitID: result.commit || '',
                    filePath: result.path,
                    disableTimeout: false,
                    format: args.format,
                    ranges: args.ranges,
                },
                false
            ),
        [result, fetchHighlightedFileLineRanges]
    )

    const fetchHighlightedFileMatchLineRanges = React.useCallback(
        (startLine: number, endLine: number) => {
            const startTime = Date.now()
            return fetchFileRangeMatches({
                format: HighlightResponseFormat.HTML_HIGHLIGHT,
                ranges: grouped.map(
                    (group): HighlightLineRange => ({
                        startLine: group.startLine,
                        endLine: group.endLine,
                    })
                ),
            }).pipe(
                map(lines => {
                    const endTime = Date.now()
                    telemetryService.log(
                        'search.latencies.frontend.code-load',
                        { durationMs: endTime - startTime },
                        { durationMs: endTime - startTime }
                    )
                    return lines[grouped.findIndex(group => group.startLine === startLine && group.endLine === endLine)]
                })
            )
        },
        [fetchFileRangeMatches, grouped, telemetryService]
    )

    const fetchPlainTextFileMatchLineRanges = React.useCallback(
        (startLine: number, endLine: number) =>
            fetchFileRangeMatches({
                format: HighlightResponseFormat.HTML_PLAINTEXT,
                ranges: grouped.map(
                    (group): HighlightLineRange => ({
                        startLine: group.startLine,
                        endLine: group.endLine,
                    })
                ),
            }).pipe(
                map(
                    lines =>
                        lines[grouped.findIndex(group => group.startLine === startLine && group.endLine === endLine)]
                )
            ),
        [fetchFileRangeMatches, grouped]
    )

    const createCodeExcerptLink = (group: MatchGroup): string => {
        const positionOrRangeQueryParameter = toPositionOrRangeQueryParameter({ position: group.position })
        return appendLineRangeQueryParameter(
            appendSubtreeQueryParameter(getFileMatchUrl(result)),
            positionOrRangeQueryParameter
        )
    }

    const codeIntelViewerUpdatesProps = useMemo(
        () =>
            grouped && result.type === 'content' && extensionsController
                ? {
                      extensionsController,
                      repositoryName: result.repository,
                      filePath: result.path,
                      revision: result.commit,
                  }
                : undefined,
        [extensionsController, result, grouped]
    )
    const viewerUpdates = useCodeIntelViewerUpdates(codeIntelViewerUpdatesProps)

    const history = useHistory()
    const navigateToFile = useCallback(
        (event: KeyboardEvent<HTMLElement> | MouseEvent<HTMLElement>): void =>
            navigateToCodeExcerpt(event, props.openInNewTab ?? false, history),
        [props.openInNewTab, history]
    )

    const logEventOnCopy = useCallback(() => {
        telemetryService.log(...codeCopiedEvent('file-match'))
    }, [telemetryService])

    const getItemRef = useFocusOnLoadedMore<HTMLDivElement>(grouped.length ?? 0)

    return (
        <div className={styles.fileMatchChildren} data-testid="file-match-children">
            {result.repoLastFetched && <LastSyncedIcon lastSyncedTime={result.repoLastFetched} />}

            {/* Line matches */}
            {grouped.length > 0 && (
                <div>
                    {grouped.map((group, index) => (
                        <div
                            key={`linematch:${getFileMatchUrl(result)}${group.position.line}:${
                                group.position.character
                            }`}
                            className={classNames('test-file-match-children-item-wrapper', styles.itemCodeWrapper)}
                        >
                            <div
                                data-href={createCodeExcerptLink(group)}
                                className={classNames(
                                    'test-file-match-children-item',
                                    styles.item,
                                    styles.itemClickable
                                )}
                                onClick={navigateToFile}
                                onMouseUp={navigateToFileOnMiddleMouseButtonClick}
                                onKeyDown={navigateToFile}
                                data-testid="file-match-children-item"
                                tabIndex={0}
                                role="link"
                                ref={getItemRef(index)}
                            >
                                <CodeExcerpt
                                    repoName={result.repository}
                                    commitID={result.commit || ''}
                                    filePath={result.path}
                                    startLine={group.startLine}
                                    endLine={group.endLine}
                                    highlightRanges={group.matches}
                                    fetchHighlightedFileRangeLines={fetchHighlightedFileMatchLineRanges}
                                    fetchPlainTextFileRangeLines={
                                        enableLazyFileResultSyntaxHighlighting
                                            ? fetchPlainTextFileMatchLineRanges
                                            : undefined
                                    }
                                    blobLines={group.blobLines}
                                    viewerUpdates={viewerUpdates}
                                    hoverifier={props.hoverifier}
                                    onCopy={logEventOnCopy}
                                />
                            </div>
                        </div>
                    ))}
                </div>
            )}
        </div>
    )
}
