package com.rentagf.notification.interfaces.event;

import static org.junit.jupiter.api.Assertions.*;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.rentagf.notification.interfaces.event.resolver.BookingCancelledResolver;
import com.rentagf.notification.interfaces.event.resolver.DisputeResolvedResolver;
import com.rentagf.notification.interfaces.event.resolver.SimpleRecipientResolver;
import java.util.List;
import java.util.Map;
import java.util.UUID;
import org.junit.jupiter.api.Tag;
import org.junit.jupiter.api.Test;

@Tag("unit")
public class RecipientResolverRegistryTest {

  private final RecipientResolverRegistry registry;

  public RecipientResolverRegistryTest() {
    ObjectMapper objectMapper = new ObjectMapper();

    SimpleRecipientResolver simpleFallbackResolver = new SimpleRecipientResolver();

    List<com.rentagf.notification.interfaces.event.resolver.RecipientResolver> list =
        List.of(
            new BookingCancelledResolver(objectMapper), new DisputeResolvedResolver(objectMapper));

    this.registry = new RecipientResolverRegistry(list, simpleFallbackResolver);
  }

  @Test
  public void testBookingCancelledByClientResolvesToCompanion() {
    UUID clientId = UUID.randomUUID();
    UUID companionId = UUID.randomUUID();

    Map<String, Object> data =
        Map.of(
            "clientId", clientId.toString(),
            "companionId", companionId.toString(),
            "actorRole", "CLIENT");

    List<UUID> recipients = registry.resolve("booking.booking-cancelled.v1", data, null);

    assertEquals(1, recipients.size());
    assertEquals(companionId, recipients.getFirst());
  }

  @Test
  public void testBookingCancelledByCompanionResolvesToClient() {
    UUID clientId = UUID.randomUUID();
    UUID companionId = UUID.randomUUID();

    Map<String, Object> data =
        Map.of(
            "clientId", clientId.toString(),
            "companionId", companionId.toString(),
            "actorRole", "COMPANION");

    List<UUID> recipients = registry.resolve("booking.booking-cancelled.v1", data, null);

    assertEquals(1, recipients.size());
    assertEquals(clientId, recipients.getFirst());
  }

  @Test
  public void testDisputeResolvedResolvesToBothParties() {
    UUID clientId = UUID.randomUUID();
    UUID companionId = UUID.randomUUID();

    Map<String, Object> data =
        Map.of(
            "clientId", clientId.toString(),
            "companionId", companionId.toString());

    List<UUID> recipients = registry.resolve("dispute.dispute-resolved.v1", data, null);

    assertEquals(2, recipients.size());
    assertTrue(recipients.contains(clientId));
    assertTrue(recipients.contains(companionId));
  }

  @Test
  public void testDefaultFieldExtractionSuccessfully() {
    UUID userId = UUID.randomUUID();
    Map<String, Object> data = Map.of("userId", userId.toString());

    List<UUID> recipients = registry.resolve("finance.kano-coin-deposited.v1", data, "userId");

    assertEquals(1, recipients.size());
    assertEquals(userId, recipients.getFirst());
  }

  @Test
  public void testMissingRecipientFieldThrowsIllegalArgumentException() {
    Map<String, Object> data = Map.of("wrongField", UUID.randomUUID().toString());
    assertThrows(
        IllegalArgumentException.class,
        () -> registry.resolve("finance.kano-coin-deposited.v1", data, "userId"));
  }
}
