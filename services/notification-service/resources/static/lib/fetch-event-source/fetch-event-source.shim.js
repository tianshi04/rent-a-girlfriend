import { fetchEventSource, EventStreamContentType } from './fetch.js';
// Expose to global scope for legacy scripts
window.fetchEventSource = fetchEventSource;
window.EventStreamContentType = EventStreamContentType;
