<script>
  // Modal reutilizável: fecha no clique no fundo e no Esc.
  let { title, onclose, children } = $props()

  function backdrop(event) {
    if (event.target === event.currentTarget) onclose()
  }

  function keydown(event) {
    if (event.key === 'Escape') onclose()
  }
</script>

<svelte:window onkeydown={keydown} />

<!-- O fundo é clicável por conveniência; o Esc cobre o acesso por teclado. -->
<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<div class="modal-bg" role="dialog" aria-modal="true" aria-label={title} onclick={backdrop}>
  <div class="modal">
    <div class="modal-head">
      <h2>{title}</h2>
      <button type="button" class="close" aria-label="Fechar" onclick={onclose}>×</button>
    </div>
    {@render children()}
  </div>
</div>
