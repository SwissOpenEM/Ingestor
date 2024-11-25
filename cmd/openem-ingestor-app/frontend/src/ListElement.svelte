<script>
  export let id;
  export let value;
  export let status;
  export let progress;
  export let extractMetadata;
  export let cancelTask;
  export let scheduleTask;
  export let removeTask;

  let metadata = "no metadata yet";
</script>

<div class="listitem">
  <div class="path">
    {value}
  </div>
  <div>
    {progress} %
  </div>
  <div class="buttongrid">
    <button
      class="btn"
      on:click={() => {
        extractMetadata(id).then((a) => (metadata = a));
      }}>Extract Metadata</button
    >
    <button
      class="btn"
      on:click={() => {
        scheduleTask(id);
      }}>Start Ingestion</button
    >
    <button
      class="btn"
      on:click={() => {
        cancelTask(id);
      }}>Cancel</button
    >
    <button
      class="btn"
      on:click={() => {
        removeTask(id);
      }}>Remove</button
    >
  </div>
  <div class="path"></div>
  <div class="textarea">
    <textarea class="textarea" bind:value={status} />
  </div>
  <div>
    <textarea class="textarea" bind:value={metadata} />
  </div>
</div>

<style>
  .listitem {
    display: grid;
    grid-template-columns: 5fr 2fr 2fr;
    grid-template-rows: 2;
    padding-right: 30px;
    border-style: solid;
    border-color: teal;
    border-radius: 10pt;
    padding: 10pt;
    margin-bottom: 2pt;
  }
  .buttongrid {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
  }
  .path {
    text-align: left;
    font-size: larger;
  }
  .textarea {
    color: teal;
    flex-direction: column;
    width: 100%;
    height: 200px;
    align-self: left;
  }
</style>
