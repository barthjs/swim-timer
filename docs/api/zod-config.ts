import { config } from 'zod';

// zod probes with new Function whether it may compile its validators, which the content security policy blocks.
// main.ts imports this module first, so the setting applies before Scalar validates anything.
config({
  jitless: true,
});
