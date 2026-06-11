package com.rentagf.notification;

import static com.tngtech.archunit.lang.syntax.ArchRuleDefinition.classes;
import static com.tngtech.archunit.lang.syntax.ArchRuleDefinition.noClasses;

import com.tngtech.archunit.core.importer.ImportOption;
import com.tngtech.archunit.junit.AnalyzeClasses;
import com.tngtech.archunit.junit.ArchTest;
import com.tngtech.archunit.lang.ArchRule;

/**
 * Unit Test đặc biệt sử dụng ArchUnit để tự động hóa việc kiểm soát cấu trúc và ranh giới kiến
 * trúc. Test này quét qua bytecode của dự án để phát hiện và ngăn chặn lập tức mọi lỗi rò rỉ (leak)
 * framework hoặc vi phạm ranh giới Hexagonal Architecture khi build dự án.
 */
@AnalyzeClasses(
    packages = "com.rentagf.notification",
    importOptions = ImportOption.DoNotIncludeTests.class)
public class ArchitectureConsistencyTest {

  /**
   * RULE 1: Lõi Application & Domain tuyệt đối KHÔNG được phép rò rỉ Web Framework (Spring Web,
   * SseEmitter). Điều này bảo vệ Core hoàn toàn thuần khiết (Pure Java) và độc lập tuyệt đối với
   * HTTP/Web.
   */
  @ArchTest
  public static final ArchRule coreShouldBeFreeOfSpringWebFramework =
      noClasses()
          .that()
          .resideInAPackage("..com.rentagf.notification.application..")
          .or()
          .resideInAPackage("..com.rentagf.notification.domain..")
          .should()
          .dependOnClassesThat()
          .resideInAnyPackage(
              "org.springframework.web..", "org.springframework.http..", "jakarta.servlet..")
          .as(
              "Application and Domain layers must not leak Spring Web/HTTP dependency (no org.springframework.web or org.springframework.http allowed).");

  /**
   * RULE 2: Lõi Application & Domain tuyệt đối KHÔNG được phép rò rỉ hạ tầng Database chi tiết
   * (JPA/Hibernate). Tương tác DB ở Core chỉ được diễn ra thông qua Domain Repository Port.
   */
  @ArchTest
  public static final ArchRule coreShouldBeFreeOfJpaDetail =
      noClasses()
          .that()
          .resideInAPackage("..com.rentagf.notification.application..")
          .or()
          .resideInAPackage("..com.rentagf.notification.domain..")
          .should()
          .dependOnClassesThat()
          .resideInAPackage("jakarta.persistence..")
          .as(
              "Application and Domain layers must not leak JPA details (no jakarta.persistence allowed).");

  /**
   * RULE 3: Lõi Domain Layer phải độc lập tuyệt đối. Domain chỉ được phụ thuộc vào chính nó và thư
   * viện Java chuẩn. Tuyệt đối không phụ thuộc vào Application, Infrastructure hay Interfaces
   * layers (Quy tắc Clean Architecture).
   */
  @ArchTest
  public static final ArchRule domainShouldBeCompletelyIndependent =
      classes()
          .that()
          .resideInAPackage("..com.rentagf.notification.domain..")
          .should()
          .onlyDependOnClassesThat()
          .resideInAnyPackage(
              "..com.rentagf.notification.domain..",
              "java..",
              "lombok.." // Lombok allowed for boilerplate reduction
              )
          .as(
              "Domain layer must be completely isolated and only depend on itself or standard Java libraries.");

  /**
   * RULE 4: Application Layer độc lập với Infrastructure và Interfaces. Application chỉ được giao
   * tiếp với bên ngoài thông qua Ports, không được phép gọi trực tiếp các Concrete Adapters.
   */
  @ArchTest
  public static final ArchRule applicationShouldNotDependOnInfrastructureOrInterfaces =
      noClasses()
          .that()
          .resideInAPackage("..com.rentagf.notification.application..")
          .should()
          .dependOnClassesThat()
          .resideInAnyPackage(
              "..com.rentagf.notification.infrastructure..",
              "..com.rentagf.notification.interfaces..")
          .as("Application layer must not depend on Infrastructure or Interfaces layers.");

  /**
   * RULE 5: Các Ports ở Application Layer bắt buộc phải là Interfaces. Đảm bảo tính lỏng lẻo (loose
   * coupling) và dễ dàng mock/test.
   */
  @ArchTest
  public static final ArchRule portsMustBeInterfaces =
      classes()
          .that()
          .resideInAPackage("..com.rentagf.notification.application.port..")
          .and()
          .haveSimpleNameNotContaining(
              "Result") // Loại trừ các lớp DTO/Result trả về của Port và Builder của chúng
          .should()
          .beInterfaces()
          .as(
              "All ports in application.port (except Result classes) must be interfaces to follow DIP (Dependency Inversion Principle).");
}
