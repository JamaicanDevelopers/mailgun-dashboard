var windowOptions = 'menubar=no,toolbar=no,location=no,directories=no,status=no,resizable=yes,scrollbars=yes'

$('.show-body').on('click', function(event) {
    event.preventDefault();
    window.open($(this).attr('href'), $(this).data('subject'), windowOptions);
});

$('.resend-message').on('click', function(event) {
    event.preventDefault();
    $.post($(this).attr('href'), {key: $(this).data('key')}, function(data) {
        window.alert(data);
    })
})
