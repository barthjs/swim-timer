/** Names the export files of the timing software the meet data is read from. */
export const MeetFile = {
  Categories: 'LSTCAT',
  Competitors: 'LSTCONC',
  Distances: 'LSTLONG',
  Events: 'LSTRACE',
  Phases: 'LSTROUND',
  StartEntries: 'LSTSTART',
  Styles: 'LSTSTYLE',
} as const;
export type MeetFile = (typeof MeetFile)[keyof typeof MeetFile];

/** Reports whether a meet file is stored. */
export type MeetFileResponse = {
  file: MeetFile;
  // The name the file is stored under, including the extension it is read with. Missing while unavailable.
  filename?: string;
  available: boolean;
};

/** The result for one uploaded file. error names why it was not stored. */
export type MeetFileUploadResponse = {
  filename: string;
  file?: MeetFile;
  error?: string;
};
