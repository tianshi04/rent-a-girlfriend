/**
 * @microsoft/fetch-event-source - Self-contained bundle (parse + fetch)
 * Source: https://github.com/Azure/fetch-event-source
 * Compiled into a single plain-JS file (no imports/exports) so it works
 * in a static HTML page without a bundler.
 *
 * Exposes to window:
 *   - window.fetchEventSource(url, options)
 *   - window.EventStreamContentType  ('text/event-stream')
 */
(function (global) {
    'use strict';

    // ─── parse ──────────────────────────────────────────────────────────────────

    const NewLine = 10;
    const CarriageReturn = 13;
    const Space = 32;
    const Colon = 58;

    async function getBytes(stream, onChunk) {
        const reader = stream.getReader();
        let result;
        while (!(result = await reader.read()).done) {
            onChunk(result.value);
        }
    }

    function getLines(onLine) {
        let buffer;
        let position;
        let fieldLength;
        let discardTrailingNewline = false;

        return function onChunk(arr) {
            if (buffer === undefined) {
                buffer = arr;
                position = 0;
                fieldLength = -1;
            } else {
                buffer = concat(buffer, arr);
            }

            const bufLength = buffer.length;
            let lineStart = 0;

            while (position < bufLength) {
                if (discardTrailingNewline) {
                    if (buffer[position] === NewLine) {
                        lineStart = ++position;
                    }
                    discardTrailingNewline = false;
                }

                let lineEnd = -1;
                for (; position < bufLength && lineEnd === -1; ++position) {
                    switch (buffer[position]) {
                        case Colon:
                            if (fieldLength < 0) {
                                fieldLength = position - lineStart;
                            }
                            break;
                        case CarriageReturn:
                            discardTrailingNewline = true;
                            // fall through
                        case NewLine:
                            lineEnd = position;
                            break;
                    }
                }

                if (lineEnd === -1) {
                    buffer = buffer.slice(lineStart);
                    position -= lineStart;
                    lineStart = 0;
                    break;
                }

                onLine(buffer.slice(lineStart, lineEnd), fieldLength);
                lineStart = position;
                fieldLength = -1;
            }

            if (lineStart === bufLength) {
                buffer = undefined;
            }
        };
    }

    function getMessages(onId, onRetry, onMessage) {
        let message = newMessage();
        const decoder = new TextDecoder();

        return function onLine(line, fieldLength) {
            if (line.length === 0) {
                // empty line = dispatch event
                onMessage(message);
                message = newMessage();
            } else if (fieldLength > 0) {
                const field = decoder.decode(line.slice(0, fieldLength));
                const valueOffset = fieldLength + (line[fieldLength + 1] === Space ? 2 : 1);
                const value = decoder.decode(line.slice(valueOffset));

                switch (field) {
                    case 'data':
                        message.data = message.data ? message.data + '\n' + value : value;
                        break;
                    case 'event':
                        message.event = value;
                        break;
                    case 'id':
                        onId(message.id = value);
                        break;
                    case 'retry':
                        const retry = parseInt(value, 10);
                        if (!isNaN(retry)) {
                            onRetry(message.retry = retry);
                        }
                        break;
                }
            }
        };
    }

    function concat(a, b) {
        const result = new Uint8Array(a.length + b.length);
        result.set(a);
        result.set(b, a.length);
        return result;
    }

    function newMessage() {
        return { data: '', event: '', id: '', retry: undefined };
    }

    // ─── fetch ──────────────────────────────────────────────────────────────────

    const EventStreamContentType = 'text/event-stream';
    const DefaultRetryInterval = 1000;
    const LastEventId = 'last-event-id';

    function fetchEventSource(input, options) {
        var _a = options || {};
        var inputSignal = _a.signal;
        var inputHeaders = _a.headers;
        var inputOnOpen = _a.onopen;
        var onmessage = _a.onmessage;
        var onclose = _a.onclose;
        var onerror = _a.onerror;
        var openWhenHidden = _a.openWhenHidden;
        var inputFetch = _a.fetch;

        // remove known keys, pass rest to fetch
        var rest = Object.assign({}, options);
        ['signal', 'headers', 'onopen', 'onmessage', 'onclose', 'onerror', 'openWhenHidden', 'fetch'].forEach(function (k) { delete rest[k]; });

        return new Promise(function (resolve, reject) {
            var headers = Object.assign({}, inputHeaders);
            if (!headers['accept']) {
                headers['accept'] = EventStreamContentType;
            }

            var curRequestController;

            function onVisibilityChange() {
                curRequestController.abort();
                if (!document.hidden) {
                    create();
                }
            }

            if (!openWhenHidden) {
                document.addEventListener('visibilitychange', onVisibilityChange);
            }

            var retryInterval = DefaultRetryInterval;
            var retryTimer = 0;

            function dispose() {
                document.removeEventListener('visibilitychange', onVisibilityChange);
                window.clearTimeout(retryTimer);
                curRequestController.abort();
            }

            if (inputSignal) {
                inputSignal.addEventListener('abort', function () {
                    dispose();
                    resolve();
                });
            }

            var fetchFn = inputFetch != null ? inputFetch : window.fetch.bind(window);
            var onopen = inputOnOpen != null ? inputOnOpen : defaultOnOpen;

            async function create() {
                curRequestController = new AbortController();
                try {
                    var fetchOptions = Object.assign({}, rest, { headers: headers, signal: curRequestController.signal });
                    var response = await fetchFn(input, fetchOptions);
                    await onopen(response);
                    await getBytes(
                        response.body,
                        getLines(
                            getMessages(
                                function (id) {
                                    if (id) { headers[LastEventId] = id; }
                                    else { delete headers[LastEventId]; }
                                },
                                function (retry) { retryInterval = retry; },
                                onmessage
                            )
                        )
                    );
                    if (onclose) onclose();
                    dispose();
                    resolve();
                } catch (err) {
                    if (!curRequestController.signal.aborted) {
                        try {
                            var interval = (onerror ? onerror(err) : null) || retryInterval;
                            window.clearTimeout(retryTimer);
                            retryTimer = window.setTimeout(create, interval);
                        } catch (innerErr) {
                            dispose();
                            reject(innerErr);
                        }
                    }
                }
            }

            create();
        });
    }

    function defaultOnOpen(response) {
        var contentType = response.headers.get('content-type');
        if (!contentType || !contentType.startsWith(EventStreamContentType)) {
            throw new Error('Expected content-type to be ' + EventStreamContentType + ', Actual: ' + contentType);
        }
    }

    // ─── Export to global scope ──────────────────────────────────────────────────
    global.fetchEventSource = fetchEventSource;
    global.EventStreamContentType = EventStreamContentType;

})(typeof window !== 'undefined' ? window : this);
